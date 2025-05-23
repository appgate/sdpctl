package appliance

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestResolveNameStatusCMD(t *testing.T) {
	tests := []struct {
		name, appliance, cli, want string
		wantErr                    bool
		wantErrOut                 *regexp.Regexp
	}{
		{
			name:      "basic name resolution status test",
			appliance: appliancepkg.TestAppliancePrimary,
			want: `Name                                                          Errors
----                                                          ------
aws://lb-tag:kubernetes.io/service-name=opsnonprod/erp-dev    []
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := httpmock.NewRegistry(t)
			hostname := "appgate.test"
			coll := appliancepkg.GenerateCollective(t, hostname, "6.3", "", []string{tt.appliance})
			appliance := coll.Appliances[tt.appliance]
			for _, stub := range coll.GenerateStubs(coll.GetAppliances(), *coll.Stats, *coll.UpgradedStats) {
				registry.RegisterStub(stub)
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
					URL:   fmt.Sprintf("http://%s:%d/admin", hostname, registry.Port),
				},
				IOOutWriter: stdout,
				Stdin:       in,
				StdErr:      stderr,
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registry.Client, nil
			}
			f.Appliance = func(c *configuration.Config) (*appliancepkg.Appliance, error) {
				api, _ := f.APIClient(c)

				a := &appliancepkg.Appliance{
					APIClient:  api,
					HTTPClient: api.GetConfig().HTTPClient,
					Token:      "",
				}
				return a, nil
			}
			args := []string{appliance.GetId()}
			if len(tt.cli) > 0 {
				args = append(args, tt.cli)
			}
			cmd := NewResolveNameStatusCmd(f)
			cmd.SetArgs(args)
			cmd.PersistentFlags().Bool("descending", false, "")
			cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewResolveNameStatusCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", tt.wantErrOut, err.Error())
				}
				return
			}
			body, err := io.ReadAll(stdout)
			if err != nil {
				t.Fatalf("unable to read stdout %s", err)
			}
			got := string(body)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewResolveNameStatusCmdJSON(t *testing.T) {
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
					URL: "/admin/appliances/0a11e7ba-4d18-4be1-bdc1-083be1411d7e/name-resolution-status",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
                                "resolutions": {
                                    "aws://lb-tag:kubernetes.io/service-name=opsnonprod/erp-dev": {
                                        "partial": false,
                                        "finals": [
                                            "3.120.51.78",
                                            "35.156.237.184"
                                        ],
                                        "partials": [
                                            "dns://all.GW-ELB-2001535196.eu-central-1.elb.amazonaws.com",
                                            "dns://all.purple-lb-1785267452.eu-central-1.elb.amazonaws.com"
                                        ],
                                        "errors": []
                                    }
                                }
                            }`))
						}
					},
				},
			},
			wantErr: false,
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
			f.Appliance = func(c *configuration.Config) (*appliancepkg.Appliance, error) {
				api, _ := f.APIClient(c)

				a := &appliancepkg.Appliance{
					APIClient:  api,
					HTTPClient: api.GetConfig().HTTPClient,
					Token:      "",
				}
				return a, nil
			}
			cmd := NewResolveNameStatusCmd(f)
			cmd.SetArgs([]string{"0a11e7ba-4d18-4be1-bdc1-083be1411d7e", "--json"})
			cmd.PersistentFlags().Bool("descending", false, "")
			cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")

			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewResolveNameStatusCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", tt.wantErrOut, err.Error())
				}
				return
			}
			body, err := io.ReadAll(stdout)
			if err != nil {
				t.Fatalf("unable to read stdout %s", err)
			}
			got := string(body)
			if !util.IsJSON(got) {
				t.Fatalf("Expected JSON output - got stdout\n%q\n", got)
			}
			want := `{
  "resolutions": {
    "aws://lb-tag:kubernetes.io/service-name=opsnonprod/erp-dev": {
      "errors": []
    }
  }
}
`
			assert.Equal(t, want, got)
		})
	}
}
