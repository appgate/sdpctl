package license

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/go-cmp/cmp"
)

func TestPruneCommand(t *testing.T) {
	tests := []struct {
		name       string
		cli        string
		httpStubs  []httpmock.Stub
		wantErr    bool
		wantErrOut *regexp.Regexp
		wantOutput string
	}{
		{
			name: "http 204",
			httpStubs: []httpmock.Stub{
				{
					URL: "/license/users/prune",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							rw.WriteHeader(http.StatusNoContent)
						}
					},
				},
			},
			wantErr:    false,
			wantOutput: "User licenses pruned\n",
		},
		{
			name: "unexpected http response",
			httpStubs: []httpmock.Stub{
				{
					URL: "/license/users/prune",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							rw.WriteHeader(http.StatusOK)
						}
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Could not prune the user licenses got HTTP 200`),
		},
		{
			name: "not found",
			httpStubs: []httpmock.Stub{
				{
					URL: "/license/users/prune",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.WriteHeader(http.StatusNotFound)
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Could not prune the user licenses, not supported on your appliance version`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := httpmock.NewRegistry(t)
			for _, v := range tt.httpStubs {
				registry.Register(v.URL, v.Responder)
			}
			defer registry.Teardown()
			registry.Serve()
			stdout := &bytes.Buffer{}
			opts := &licenseOpts{
				Out:     stdout,
				BaseURL: fmt.Sprintf("http://127.0.0.1:%d", registry.Port),
				HTTPClient: func() (*http.Client, error) {
					return &http.Client{}, nil
				},
			}
			cmd := NewPruneCmd(opts)
			_, err := cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewPruneCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Fatalf("Expected output to match, got:\n%s\n expected: \n%s\n", tt.wantErrOut, err.Error())
				}

			}
			got, err := io.ReadAll(stdout)
			if err != nil {
				t.Fatalf("unable to read stdout %s", err)
			}
			gotStr := string(got)
			if diff := cmp.Diff(tt.wantOutput, gotStr); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
