package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)


func createRequest(method, path string) *http.Request{
	r, _ := http.NewRequest(method, path, nil)
	return r
}

func TestServer_mainHandler(t *testing.T) {
	type fields struct {
		Port            int
		websocket       *websocket.Conn
		pendingRequests map[uuid.UUID]*Request
	}
	tests := []struct {
		name   string
		fields fields
		expectedCode int
		testRequest *http.Request
	}{
		{
			name:         "no websocket client",
			fields:       fields{}, // empty
			expectedCode: http.StatusBadGateway,
			testRequest:  createRequest("GET", "/test"),
		},
		//{
		//	name: "send request to client",
		//	fields: fields{
		//		pendingRequests: make(map[uuid.UUID]*Request),
		//	},
		//	testRequest:  createRequest("GET", "/test"),
		//// test that websocket.writemessage was done
		//},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Server{
				Port:            tt.fields.Port,
				websocket:       tt.fields.websocket,
				pendingRequests: tt.fields.pendingRequests,
			}

			w := httptest.NewServer(http.HandlerFunc(s.mainHandler))
			defer w.Close()

			rr := httptest.NewRecorder()
			http.HandlerFunc(s.mainHandler).ServeHTTP(rr, tt.testRequest)


			if tt.expectedCode != 0{
				assert.Equal(t, tt.expectedCode, rr.Code)
			}
		})
	}
}