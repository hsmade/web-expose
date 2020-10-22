package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/hsmade/web-expose/pkg/server"
)

func TestClient_prepareRequest(t *testing.T) {
	body1 := []byte("my body is over the ocean")

	type fields struct {
		LocalServer string
		Scheme      string
	}
	type args struct {
		request *server.ForwardRequest
	}
	type result struct {
		Method string
		URL    string
		Body   []byte
		Header http.Header
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   result
		wantErr bool
	}{
		{
			name: "happy path: ",
			fields: fields{
				LocalServer: "local",
				Scheme:      "https",
			},
			args: args{
				request: &server.ForwardRequest{
					Header: http.Header{"key": []string{"value"}},
					URI: &url.URL{
						Scheme: "http",
						Host:   "google.com",
						Path:   "/mypath",
					},
					Body:   &body1,
					Method: "GET",
				},
			},
			want: result{
				Method: "GET",
				URL:    "https://local/mypath",
				Body:   body1,
				Header: http.Header{"key": []string{"value"}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				LocalServer: tt.fields.LocalServer,
				Scheme:      tt.fields.Scheme,
			}
			got, err := c.prepareRequest(tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != nil {
				body, _ := ioutil.ReadAll(got.Body)
				gotSimple := result{
					Method: got.Method,
					URL:    got.URL.String(),
					Body:   body,
					Header: got.Header,
				}
				if !cmp.Equal(gotSimple, tt.want) {
					t.Errorf("prepareRequest() got = %v, want %v", gotSimple, tt.want)
				}
			}
		})
	}
}

func TestClient_prepareResponse(t *testing.T) {
	body1 := []byte("my body is over the ocean")

	type args struct {
		response *http.Response
		id       uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		want    *server.Response
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				response: &http.Response{
					StatusCode: 500,
					Header: http.Header{"key": []string{"value"}},
					Body: ioutil.NopCloser(bytes.NewReader(body1)),
				},
				id:       uuid.UUID{0x01},
			},
			want: &server.Response{
				ID:      uuid.UUID{0x01},
				Headers: http.Header{"key": []string{"value"}},
				Status:  500,
				Body:    &body1,
			},
			wantErr: false,
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{}
			got, err := c.prepareResponse(tt.args.response, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareResponse() got = %v, want %v", got, tt.want)
			}
		})
	}
}