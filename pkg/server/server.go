package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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
	ID uuid.UUID
	Header http.Header
	URI	*url.URL
	Body *[]byte
	Method string
	Proto string
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
		var response Response
		err := conn.ReadJSON(&response)
		if err != nil {
			logrus.WithError(err).WithField("websocket-client", conn.RemoteAddr().String()).Error("failed to receive message from websocket")
			break
		}
		logrus.WithFields(logrus.Fields{
			"websocket-client": conn.RemoteAddr().String(),
			"ID": response.ID,
		}).Debug("received response")

		originalRequest, ok := s.pendingRequests[response.ID]
		if !ok {
			logrus.WithField("websocket-client", conn.RemoteAddr().String()).WithField("ID", response.ID).Error("no response channel")
		}
		originalRequest <- &response
	}
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

	request := newRequest(w, r)
	responseChan := make(chan *Response)
	s.pendingRequests[request.ID] = responseChan

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.WithFields(logFields).WithError(err).Error("failed to read body from request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.websocket.WriteJSON(ForwardRequest{
		ID: request.ID,
		Header: r.Header.Clone(),
		Body: &body,
		Method: r.Method,
		URI: r.URL,
	})
	if err != nil {
		logrus.WithError(err).WithFields(logFields).WithField("request", request).Error("Could not send request to websocket")
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	response := <- responseChan
	logrus.WithFields(logFields).WithField("ID", response.ID).Debug("received response")

	copyHeader(w.Header(), response.Headers)
	w.WriteHeader(response.Status)
	_, err = w.Write(*response.Body)
	if err != nil {
		logrus.WithError(err).WithFields(logFields).WithField("request", request).Error("Could not send response to client")
	}
	delete(s.pendingRequests, request.ID)
	logrus.WithFields(logFields).Debug("done handling request")
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}