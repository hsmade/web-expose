package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Server struct {
	Port            int
	websocket       *websocket.Conn
	pendingRequests map[uuid.UUID]chan *Response
}

type Request struct {
	R  *http.Request
	w  http.ResponseWriter
	ID uuid.UUID
}

type Response struct {
	ID      uuid.UUID
	Headers http.Header
	Status  int
	Body    *[]byte
}
type ForwardRequest struct {
	ID     uuid.UUID
	Header http.Header
	URI    *url.URL
	Body   *[]byte
	Method string
	Proto  string
}

func newRequest(w http.ResponseWriter, r *http.Request) Request {
	return Request{
		ID: uuid.New(),
		R:  r,
		w:  w,
	}
}

func NewServer(port int) *Server {
	return &Server{
		Port:            port,
		pendingRequests: make(map[uuid.UUID]chan *Response),
	}
}

// Run the webserver
func (s *Server) Run() error {
	http.HandleFunc("/", s.mainHandler)
	http.HandleFunc("/WSconnect", s.websocketHandler)
	logrus.Infof("started webserver on port %d", s.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil)
}

// receive a new websocket connection and store it in the server struct
func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	logrus.WithField("remote-ip", r.RemoteAddr).Info("new websocket client")
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).WithField("remote-ip", r.RemoteAddr).Error("upgrade failed")
		return
	}
	defer conn.Close()

	// store the connection, so others can write
	s.websocket = conn

	// read responses
	for {
		s.handleWebsocketResponse()
	}
}

func (s *Server) handleWebsocketResponse() {
	var response Response
	err := s.websocket.ReadJSON(&response)
	if err != nil {
		logrus.WithError(err).WithField("websocket-client", s.websocket.RemoteAddr().String()).Error("failed to receive message from websocket")
		return
	}

	logrus.WithFields(logrus.Fields{
		"websocket-client": s.websocket.RemoteAddr().String(),
		"ID":               response.ID,
	}).Debug("received response")

	originalRequest, ok := s.pendingRequests[response.ID]
	if !ok {
		logrus.WithField("websocket-client", s.websocket.RemoteAddr().String()).WithField("ID", response.ID).Error("no response channel")
	}
	originalRequest <- &response

}

// receive a new web request, register it, and forward it to the websocket client
func (s *Server) mainHandler(w http.ResponseWriter, r *http.Request) {
	logFields := logrus.Fields{
		"remote_ip": r.RemoteAddr,
		"url":       r.RequestURI,
		"method":    r.Method,
	}

	logrus.WithFields(logFields).Debug("received request")

	if s.websocket == nil {
		logrus.WithFields(logFields).Error("there is no websocket to forward to")
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	forwardRequest, err := s.prepareRequest(r)
	if err != nil {
		logrus.WithFields(logFields).WithError(err).Error("failed to prepare request for forwarding")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// create a chan to watch for the response
	responseChan := make(chan *Response)
	s.pendingRequests[forwardRequest.ID] = responseChan

	// send it to the remote site
	err = s.websocket.WriteJSON(forwardRequest)
	if err != nil {
		logrus.WithError(err).WithFields(logFields).WithField("request", forwardRequest).Error("Could not send request to websocket")
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// wait for a response
	response := <-responseChan
	logrus.WithFields(logFields).WithField("ID", response.ID).Debug("received response")

	// send the response back to the user
	copyHeader(w.Header(), response.Headers)
	w.WriteHeader(response.Status)
	_, err = w.Write(*response.Body)
	if err != nil {
		logrus.WithError(err).WithFields(logFields).WithField("request", forwardRequest).Error("Could not send response to client")
	}

	// cleanup
	delete(s.pendingRequests, forwardRequest.ID)
	logrus.WithFields(logFields).Debug("done handling request")
}

func (s *Server) prepareRequest(r *http.Request) (*ForwardRequest, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read body from request")
	}

	return &ForwardRequest{
		ID:     uuid.New(),
		Header: r.Header.Clone(),
		Body:   &body,
		Method: r.Method,
		URI:    r.URL,
	}, nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
