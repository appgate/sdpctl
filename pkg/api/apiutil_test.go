package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHTTPErrorResponse(t *testing.T) {
	type args struct {
		response *http.Response
		err      error
	}
	tests := []struct {
		name        string
		args        args
		errorString string
	}{
		{
			name: "no response",
			args: args{
				response: nil,
				err:      errors.New("aa"),
			},
			errorString: "No response aa",
		},
		{
			name: "HTTP 500 no json",
			args: args{
				response: &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				err: errors.New("bb"),
			},
			errorString: "HTTP 500 - bb",
		},
		{
			name: "HTTP 400 invalid json",
			args: args{
				response: &http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader(`{"not": "expected"}`)),
				},
				err: errors.New("cc"),
			},
			errorString: "HTTP 400 - cc",
		},
		{
			name: "HTTP 400 invalid json format",
			args: args{
				response: &http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader(`{"not": "incomplete..`)),
				},
				err: errors.New("something strange"),
			},
			errorString: "HTTP 400 - something strange",
		},
		{
			name: "HTTP 400 expected json format",
			args: args{
				response: &http.Response{
					StatusCode: 400,
					Body: io.NopCloser(strings.NewReader(`{
                        "id": "abc",
                        "message": "internal error message",
                        "errors": [
                            {
                                "field": "field 1",
                                "message": "hello"
                            },
                            {
                                "field": "field 2",
                                "message": "world"
                            }
                        ]
                    }`)),
				},
				err: errors.New("cc"),
			},
			errorString: "HTTP 400 - cc",
		},
		{
			name: "HTTP 422 expected json format no errors array",
			args: args{
				response: &http.Response{
					StatusCode: 422,
					Body: io.NopCloser(strings.NewReader(`{
		                "id": "abc",
		                "message": "internal error message"
		            }`)),
				},
				err: errors.New("cc"),
			},
			errorString: "HTTP 422 - cc",
		},
		{
			name: "HTTP 403 no error",
			args: args{
				response: &http.Response{
					StatusCode: 403,
					Body: io.NopCloser(strings.NewReader(`{
                        "id": "forbidden",
                        "message": "You don't have permission to access this resource."
                    }`)),
				},
				err: nil,
			},
			errorString: "You don't have permission to access this resource.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HTTPErrorResponse(tt.args.response, tt.args.err)
			if err == nil {
				t.Fatal("HTTPErrorResponse must return err, got nil")
			}
			if err.Error() != tt.errorString {
				t.Errorf("expected %q got %q", tt.errorString, err.Error())
			}
		})
	}
}
