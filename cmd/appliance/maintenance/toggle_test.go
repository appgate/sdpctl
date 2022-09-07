package maintenance

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/util"
)

func TestNewToggleCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		httpStubs  []httpmock.Stub
		wantErr    bool
		wantErrOut *regexp.Regexp
		version    int
		wantJSON   bool
	}{
		{
			name: "two arguments no interactive json format",
			args: []string{"20e75a08-96c6-4ea3-833e-cdbac346e2ae", "true", "--no-interactive", "--json"},
			httpStubs: []httpmock.Stub{
				{
					URL: "/appliances/20e75a08-96c6-4ea3-833e-cdbac346e2ae/maintenance",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
		                        "id": "1ed6bc6a-9e6f-4d74-bdc5-a76e2a2b49e6"
		                    }`))
						}
					},
				},
				{
					URL: "/appliances/20e75a08-96c6-4ea3-833e-cdbac346e2ae/change/1ed6bc6a-9e6f-4d74-bdc5-a76e2a2b49e6",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
		                        "id": "1ed6bc6a-9e6f-4d74-bdc5-a76e2a2b49e6",
		                        "result": "success",
		                        "status": "completed"
		                    }`))
						}
					},
				},
			},
			wantErr:  false,
			version:  15,
			wantJSON: true,
		},
		{
			name:       "version not supported",
			args:       []string{"20e75a08-96c6-4ea3-833e-cdbac346e2ae", "true", "--no-interactive", "--json"},
			httpStubs:  []httpmock.Stub{},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`maintenance mode is not supported on this version`),
			version:    13,
			wantJSON:   false,
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
			stdin := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			in := io.NopCloser(stdin)
			f := &factory.Factory{
				Config: &configuration.Config{
					Debug:   false,
					URL:     fmt.Sprintf("http://localhost:%d", registry.Port),
					Version: tt.version,
				},
				IOOutWriter: stdout,
				Stdin:       in,
				StdErr:      stderr,
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registry.Client, nil
			}
			f.Appliance = func(c *configuration.Config) (*appliance.Appliance, error) {
				api, _ := f.APIClient(c)

				a := &appliance.Appliance{
					APIClient:  api,
					HTTPClient: api.GetConfig().HTTPClient,
					Token:      "",
				}
				return a, nil
			}
			cmd := NewToggleCmd(f)
			cmd.PersistentFlags().Bool("no-interactive", false, "suppress interactive prompt with auto accept")
			cmd.SetArgs(tt.args)

			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewToggleCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", tt.wantErrOut, err.Error())
				}
				return
			}
			got, err := io.ReadAll(stdout)
			if err != nil {
				t.Fatalf("unable to read stdout %s", err)
			}
			if tt.wantJSON {
				if !util.IsJSON(string(got)) {
					t.Fatalf("Expected JSON output - got stdout\n%q\n", string(got))
				}
			}

		})
	}
}
