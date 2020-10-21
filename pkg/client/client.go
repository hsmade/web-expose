package client

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/hsmade/web-expose/pkg/server"
)

type Client struct {
	RemoteUrl   string // ws://host:port/path
	LocalServer string // host:port
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

		response, err := c.doRequest(&request)
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

func (c *Client) doRequest(request *server.ForwardRequest) (*server.Response, error) {
	response := server.Response{
		ID: request.ID,
	}


	request.URI.Host = c.LocalServer
	request.URI.Scheme = "http"
	logrus.Debugf("url to request: %s",request.URI.String())
	r, err := http.NewRequest(request.Method, request.URI.String(), bytes.NewReader(*request.Body))
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to do request")
	}

	response.Status = res.StatusCode
	response.Headers = res.Header.Clone()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed read body from response")
	}
	response.Body = &body
	return &response, nil
}
