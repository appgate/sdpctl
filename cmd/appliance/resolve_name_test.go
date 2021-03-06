package appliance

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

func TestNewResolveNameCmdJSON(t *testing.T) {
	tests := []struct {
		name       string
		cli        string
		httpStubs  []httpmock.Stub
		wantErr    bool
		wantErrOut *regexp.Regexp
	}{
		{
			name: "test JSON 200",
			httpStubs: []httpmock.Stub{
				{
					URL: "/appliances/0a11e7ba-4d18-4be1-bdc1-083be1411d7e/test-resolver-name",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
		                        "resourceName": "aws://tag:Application=Software Defined Perimeter"
		                    }
		                    `))
						}
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test JSON 422",
			httpStubs: []httpmock.Stub{
				{
					URL: "/appliances/0a11e7ba-4d18-4be1-bdc1-083be1411d7e/test-resolver-name",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusUnprocessableEntity)
							fmt.Fprint(rw, string(`{
                                "id": "string",
                                "message": "string",
                                "errors": [
                                  {
                                    "field": "name",
                                    "message": "may not be null"
                                  }
                                ]
                            }`))
						}
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`name may not be null`),
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
					Debug: false,
					URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
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
			cmd := NewResolveNameCmd(f)
			cmd.SetArgs([]string{"0a11e7ba-4d18-4be1-bdc1-083be1411d7e", "--json", "dns://appgate.com"})

			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewResolveNameCmd() error = %v, wantErr %v", err, tt.wantErr)
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
			if !util.IsJSON(string(got)) {
				t.Fatalf("Expected JSON output - got stdout\n%q\n", string(got))
			}
		})
	}
}
