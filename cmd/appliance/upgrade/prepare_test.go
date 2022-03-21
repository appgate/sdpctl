package upgrade

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

type mockUpgradeStatus struct{}

func (u *mockUpgradeStatus) Wait(timeout time.Duration, appliances []openapi.Appliance, desiredStatus string) error {
	return nil
}

type errorUpgradeStatus struct{}

func (u *errorUpgradeStatus) Wait(timeout time.Duration, appliances []openapi.Appliance, desiredStatus string) error {
	return fmt.Errorf("gateway never reached %s, got failed", desiredStatus)
}

func NewApplianceCmd(f *factory.Factory) *cobra.Command {
	// define prepare parent command flags so we can include these in the tests.
	cmd := &cobra.Command{
		Use:              "appliance",
		Short:            "interact with appliances",
		Aliases:          []string{"app", "a"},
		TraverseChildren: true,
	}
	cmd.PersistentFlags().Bool("no-interactive", false, "suppress interactive prompt with auto accept")
	cmd.PersistentFlags().StringToStringP("filter", "f", map[string]string{}, "")
	cmd.PersistentFlags().StringToStringP("exclude", "e", map[string]string{}, "Exclude appliances. Adheres to the same syntax and key-value pairs as '--filter'")
	return cmd
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
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1.img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true) // auto-scaling warning
				s.StubOne(true) // disk usage
				s.StubOne(true) // peer_warning message
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
					URL:       "/files/appgate-5.5.1.img.zip",
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
			name: "with gateway filter",
			cli:  `upgrade prepare --filter function=gateway --image './testdata/appgate-5.5.1.img.zip'`,
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true) // auto-scaling warning
				s.StubOne(true) // disk usage
				s.StubOne(true) // peer_warning message
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
					URL:       "/files/appgate-5.5.1.img.zip",
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
			name:       "invalid throttle",
			cli:        "upgrade prepare --image './testdata/appgate-5.5.1.img.zip' --throttle 0",
			wantErr:    true,
			wantErrOut: regexp.MustCompile("Prepare failed: throttle too small"),
		},
		{
			name:                "error upgrade status",
			cli:                 "upgrade prepare --image './testdata/appgate-5.5.1.img.zip'",
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`Timeout exceeded when waiting for correct appliance upgrade state. See log for details`),
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true) // auto-scaling warning
				s.StubOne(true) // disk usage
				s.StubOne(true) // peer_warning message
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
					URL:       "/files/appgate-5.5.1.img.zip",
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
			cli:        "upgrade prepare",
			httpStubs:  []httpmock.Stub{},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`--image is mandatory`),
		},
		{
			name: "timeout flag",
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1.img.zip' --timeout 0s",
			askStubs: func(as *prompt.AskStubber) {
				as.StubOne(true) // auto-scaling warning
				as.StubOne(true) // disk usage
				as.StubOne(true) // peer_warning message
				as.StubOne(true) // upgrade_confirm
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
					URL:       "/files/appgate-5.5.1.img.zip",
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
			wantErr:    true,
			wantErrOut: regexp.MustCompile("context deadline exceeded"),
		},
		{
			name: "disagree with peer warning",
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1.img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true)  // auto-scaling warning
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
			name: "no prepare confirmation",
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1.img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true)  // auto-scaling warning
				s.StubOne(true)  // disk usage
				s.StubOne(true)  // peer_warning message
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
			cli:  "upgrade prepare --image 'abc123456'",
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
		{
			name: "file name error",
			cli:  "upgrade prepare --image './testdata/appgate.img'",
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
			wantErrOut: regexp.MustCompile(`Invalid mimetype on image file. The format is expected to be a .img.zip archive.`),
		},
		{
			name: "invalid zip file error",
			cli:  "upgrade prepare --image './testdata/invalid-5.5.1.img.zip'",
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
			wantErrOut: regexp.MustCompile(`zip: not a valid zip file`),
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
			// add parent command to allow us to include test with parent flags
			cmd := NewApplianceCmd(f)
			upgradeCmd := NewUpgradeCmd(f)
			cmd.AddCommand(upgradeCmd)
			upgradeCmd.AddCommand(NewPrepareUpgradeCmd(f))

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

func TestCheckImageFilename(t *testing.T) {
	type args struct {
		i string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "s3 bucket url",
			args: args{
				i: "https://s3.us-central-1.amazonaws.com/bucket/appgate-5.5.99-123-release.img.zip",
			},
			wantErr: false,
		},

		{
			name: "localpath",
			args: args{
				i: "/tmp/artifacts/55/appgate-5.5.2-99999-release.img.zip",
			},
			wantErr: false,
		},
		{
			name: "test url with get variables",
			args: args{
				i: "https://download.com/release-5.5/artifact/appgate-5.5.3-27278-release.img.zip?is-build-type-id",
			},
			wantErr: false,
		},
		{
			name: "test url with get variables key value",
			args: args{
				i: "https://download.com/release-5.5/artifact/appgate-5.5.3-27278-release.img.zip?foo=bar",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkImageFilename(tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("checkImageFilename() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
