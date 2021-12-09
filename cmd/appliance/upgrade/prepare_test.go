package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/AlecAivazis/survey/v2/core"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/shlex"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

type mockUpgradeStatus struct{}

func (u *mockUpgradeStatus) Wait(ctx context.Context, appliances []openapi.Appliance, desiredStatus string) error {
	return nil
}

type errorUpgradeStatus struct{}

func (u *errorUpgradeStatus) Wait(ctx context.Context, appliances []openapi.Appliance, desiredStatus string) error {
	return fmt.Errorf("gateway never reached %s, got failed", desiredStatus)
}

func TestUpgradePrepareCommand(t *testing.T) {

	tests := []struct {
		name                string
		cli                 string
		askStubs            func(*prompt.AskStubber)
		httpStubs           []httpmock.Stub
		upgradeStatusWorker appliancepkg.WaitForUpgradeStatus
		wantErr             bool
		wantErrOut          *regexp.Regexp
	}{
		{
			name: "with existing file",
			cli:  "prepare --image './testdata/img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true) // auto-scaling warning
				s.StubOne(true) // disk usage
				s.StubOne(true) // peer_warning message
				s.StubOne(true) // backup confirmation
				s.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_idle.json"),
				},
				{
					URL:       "/files/img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{
		                    "status": "idle",
		                    "details": "a reboot is required for the Upgrade to go into effect"
		                  }`))
					},
				},
			},
			wantErr: false,
		},
		{
			name:                "error upgrade status",
			cli:                 "prepare --image './testdata/img.zip'",
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`gateway never reached ready, got failed`),
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true) // auto-scaling warning
				s.StubOne(true) // disk usage
				s.StubOne(true) // peer_warning message
				s.StubOne(true) // backup confirmation
				s.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_idle.json"),
				},
				{
					URL:       "/files/img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{
		                    "status": "idle",
		                    "details": "a reboot is required for the Upgrade to go into effect"
		                  }`))
					},
				},
			},
			wantErr: true,
		},
		{
			name:       "no image argument",
			cli:        "prepare",
			httpStubs:  []httpmock.Stub{},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Image is mandatory`),
		},
		{
			name: "disagree with peer warning",
			cli:  "prepare --image './testdata/img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true)  // auto-scaling warning
				s.StubOne(true)  // disk usage
				s.StubOne(false) // peer_warning message
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Cancelled by user`),
		},
		{
			name: "no backup confirmation",
			cli:  "prepare --image './testdata/img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true)  // auto-scaling warning
				s.StubOne(true)  // disk usage
				s.StubOne(true)  // peer_warning message
				s.StubOne(false) // backup confirmation
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
			},
			wantErr: true,
		},
		{
			name: "no prepare confirmation",
			cli:  "prepare --image './testdata/img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true)  // auto-scaling warning
				s.StubOne(true)  // disk usage
				s.StubOne(true)  // peer_warning message
				s.StubOne(true)  // backup confirmation
				s.StubOne(false) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
			},
			wantErr: true,
		},
		{
			name: "image file not found",
			cli:  "prepare --image 'abc123456'",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Image file not found "abc123456"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			registry := httpmock.NewRegistry()
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
					Debug:                    false,
					URL:                      fmt.Sprintf("http://localhost:%d", registry.Port),
					PrimaryControllerVersion: "5.3.4-24950",
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
				if tt.upgradeStatusWorker != nil {
					a.UpgradeStatusWorker = tt.upgradeStatusWorker
				} else {
					a.UpgradeStatusWorker = new(mockUpgradeStatus)
				}

				return a, nil
			}
			cmd := NewPrepareUpgradeCmd(f)
			// cobra hack
			cmd.Flags().BoolP("help", "x", false, "")

			argv, err := shlex.Split(tt.cli)
			if err != nil {
				panic("Internal testing error, failed to split args")
			}
			cmd.SetArgs(argv)

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			stubber, teardown := prompt.InitAskStubber()
			defer teardown()

			if tt.askStubs != nil {
				tt.askStubs(stubber)
			}
			_, err = cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Logf("Stdout: %s", stdout)
				t.Fatalf("TestUpgradePrepareCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Logf("Stdout: %s", stdout)
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", err.Error(), tt.wantErrOut)
				}
			}
		})
	}
}
