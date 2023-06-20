package cmdutil

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/appgate/sdpctl/pkg/api"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

func TestCommandErrorHandling(t *testing.T) {
	type args struct {
		cmd *cobra.Command
	}
	tests := []struct {
		name         string
		args         args
		want         ExitCode
		wantedOutput string
	}{
		{
			name: "test no error",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error { return nil }},
			},
			want: ExitOK,
		},
		{
			name: "auth error",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error { return ErrExitAuth }},
			},
			want: ExitAuth,
			wantedOutput: `1 error occurred:
	* no authentication


`,
		},
		{
			name: "execution canceled by user",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error { return ErrExecutionCanceledByUser }},
			},
			want: ExitCancel,
			wantedOutput: `1 error occurred:
	* Cancelled by user


`,
		},
		{
			name: "context DeadlineExceeded error",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
					return context.DeadlineExceeded
				}},
			},
			want: ExitError,
			wantedOutput: `2 errors occurred:
	* context deadline exceeded
	* Command timed out


`,
		},
		{
			name: "ssl error",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
					return x509.UnknownAuthorityError{}
				}},
			},
			want: ExitError,
			wantedOutput: `2 errors occurred:
	* x509: certificate signed by unknown authority
	* Trust the certificate or import a PEM file using 'sdpctl configure --pem=<path/to/pem>'


`,
		},
		{
			name: "test wrapped api error",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
					response := &http.Response{
						StatusCode: http.StatusForbidden,
						Request: &http.Request{
							Method: http.MethodGet,
							URL: &url.URL{
								Scheme: "https",
								Host:   "controller.appgate.com",
								Path:   "admin/global-settings",
							},
							Close: true,
						},
					}
					response.Body = io.NopCloser(strings.NewReader(`{
                        "id": "abc",
                        "message": "internal error message"
                    }`))
					ae := api.HTTPErrorResponse(response, errors.New("foobar"))
					return fmt.Errorf("hello world %w", ae)
				}},
			},
			want: ExitError,
			wantedOutput: `4 errors occurred:
	* Run 'sdpctl privileges' to see your current user privileges
	* HTTP GET https://controller.appgate.com/admin/global-settings
	* internal error message
	* hello world HTTP 403 - foobar


`,
		},
		{
			name: "http 503 no json response body",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
					response := &http.Response{StatusCode: http.StatusBadGateway}
					response.Body = io.NopCloser(strings.NewReader(`<html>
                    <head>
                      <title>502 Bad Gateway</title>
                    </head>
                    <body>
                      <center>
                        <h1>502 Bad Gateway</h1>
                      </center>
                      <hr>
                      <center>nginx</center>
                    </body>
                    </html>`))
					return api.HTTPErrorResponse(response, errors.New("502 Bad Gateway"))
				}},
			},
			want: ExitError,
			wantedOutput: `1 error occurred:
	* HTTP 502 - 502 Bad Gateway


`,
		},
		{
			name: "api error",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
					response := &http.Response{StatusCode: 500}
					response.Body = io.NopCloser(strings.NewReader(`{
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
		                    }`))
					return api.HTTPErrorResponse(response, errors.New("foobar"))
				}},
			},
			want: ExitError,
			wantedOutput: `3 errors occurred:
	* internal error message
	* field 1 hello
	* field 2 world


`,
		},
		{
			name: "nested multierror",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
					var result error
					result = multierror.Append(result, errors.New("aa"))
					result = multierror.Append(result, errors.New("bb"))

					return result
				}},
			},
			want: ExitError,
			wantedOutput: `2 errors occurred:
	* aa
	* bb


`,
		},
		{
			name: "wrapped multierror",
			args: args{
				cmd: &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
					var result error
					result = multierror.Append(result, errors.New("golang"))
					result = multierror.Append(result, errors.New("python"))

					return fmt.Errorf("root message %w", result)
				}},
			},
			want: ExitError,
			wantedOutput: `2 errors occurred:
	* golang
	* python


`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			cmd := tt.args.cmd
			cmd.SetOut(io.Discard)
			cmd.SilenceErrors = true
			cmd.SetErr(stdout)
			if got := ExecuteCommand(tt.args.cmd); got != tt.want {
				t.Errorf("executeCommand() = %+v, want %+v", got, tt.want)
			}

			if diff := cmp.Diff(tt.wantedOutput, stdout.String()); diff != "" {
				t.Fatalf("Diff (-want +got):\n%s", diff)
			}
		})
	}
}
