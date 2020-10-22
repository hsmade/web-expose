package client

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/hsmade/web-expose/pkg/server"
)

type Client struct {
	RemoteUrl   string // ws://host:port/path
	LocalServer string // host:port
	Scheme		string // http or https
	websocket   *websocket.Conn
}

func (c *Client) Run() {
	ws, _, err := websocket.DefaultDialer.Dial(c.RemoteUrl, nil)
	c.websocket = ws
	if err != nil {
		logrus.WithError(err).WithField("remote-url", c.RemoteUrl).Fatal("failed to connect to websocket")
	}
	logrus.WithField("remote-url", c.RemoteUrl).Info("connected")

	for {
		var request server.ForwardRequest
		err = c.websocket.ReadJSON(&request)
		if err != nil {
			logrus.WithError(err).WithField("remote-url", c.RemoteUrl).Error("failed to read request")
			continue
		}

		response, err := c.doWebRequest(&request)
		if err != nil {
			logrus.WithError(err).WithField("remote-url", c.RemoteUrl).Error("failed to execute request")
			continue
		}

		err = c.websocket.WriteJSON(response)
		if err != nil {
			logrus.WithError(err).WithField("remote-url", c.RemoteUrl).Error("failed to send response")
			continue
		}
	}
}

func (c *Client) doWebRequest(forwardRequest *server.ForwardRequest) (*server.Response, error) {
	request, err := c.prepareRequest(forwardRequest)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		logrus.WithError(err).Error("failed to do request")
		return nil, errors.Wrap(err, "failed to do request")
	}

	return c.prepareResponse(response, forwardRequest.ID)
}

// create an http.Request from a forwarded request
func (c *Client) prepareRequest(request *server.ForwardRequest) (*http.Request, error) {
	request.URI.Host = c.LocalServer
	request.URI.Scheme = c.Scheme

	r, err := http.NewRequest(request.Method, request.URI.String(), bytes.NewReader(*request.Body))
	if err != nil {
		return nil, errors.Wrap(err, "creating http.Request from server.ForwardRequest")
	}
	r.Header = request.Header.Clone()
	return r, nil
}

func (c *Client) prepareResponse(response *http.Response, id uuid.UUID) (*server.Response, error) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.WithError(err).Error("failed to read body from response")

		return nil, errors.Wrap(err, "failed read body from response")
	}

	return &server.Response{
		ID: id,
		Status: response.StatusCode,
		Headers: response.Header.Clone(),
		Body: &body,
	}, nil
}