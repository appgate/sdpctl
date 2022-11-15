package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

type mockUpgradeStatus struct{}

func (u *mockUpgradeStatus) WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) error {
	return nil
}

type errorUpgradeStatus struct{}

func (u *errorUpgradeStatus) WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) error {
	return fmt.Errorf("gateway never reached %s, got failed", strings.Join(desiredStatuses, ", "))
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
	cmd.PersistentFlags().Bool("ci-mode", false, "ci mode")
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
		wantOut             *regexp.Regexp
		wantErr             bool
		wantErrOut          *regexp.Regexp
	}{
		{
			name: "with existing file",
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1-9876.img.zip'",
			askStubs: func(s *prompt.AskStubber) {
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
					URL:       "/files/appgate-5.5.1-9876.img.zip",
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
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with gateway filter",
			cli:  `upgrade prepare --filter function=gateway --image './testdata/appgate-5.5.1-9876.img.zip'`,
			askStubs: func(s *prompt.AskStubber) {
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
					URL:       "/files/appgate-5.5.1-9876.img.zip",
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
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
			},
			wantErr: false,
		},
		{
			name:                "error upgrade status",
			cli:                 "upgrade prepare --image './testdata/appgate-5.5.1-9876.img.zip'",
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`gateway never reached verifying, ready, got failed`),
			askStubs: func(s *prompt.AskStubber) {
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
					URL:       "/files/appgate-5.5.1-9876.img.zip",
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
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
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
			name: "disagree with peer warning",
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1-9876.img.zip'",
			askStubs: func(s *prompt.AskStubber) {
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
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1-9876.img.zip'",
			askStubs: func(s *prompt.AskStubber) {
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
			name:       "image file not found",
			cli:        "upgrade prepare --image 'abc123456.img.zip'",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`.+Image file not found ".+abc123456.img.zip"`),
		},
		{
			name:       "file name error",
			cli:        "upgrade prepare --image './testdata/appgate.img'",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Invalid name on image file. The format is expected to be a .img.zip archive`),
		},
		{
			name:       "invalid zip file error",
			cli:        "upgrade prepare --image './testdata/invalid.img.zip'",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`zip: not a valid zip file`),
		},
		{
			name: "prepare same version",
			cli:  "upgrade prepare --image './testdata/appgate-5.5.1-12345.img.zip'",
			askStubs: func(as *prompt.AskStubber) {
				as.StubOne(true)
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance_5.5.1.json"),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`No appliances to prepare for upgrade. All appliances may have been filtered or are already prepared. See the log for more details`),
		},
		{
			name: "force prepare same version",
			cli:  "upgrade prepare --force --image './testdata/appgate-5.5.1-12345.img.zip'",
			askStubs: func(as *prompt.AskStubber) {
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance_5.5.1.json"),
				},
				{
					URL:       "/files/appgate-5.5.1-12345.img.zip",
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
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-5.5.1-9876.img.zip"}`))
					},
				},
			},
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
					URL:     fmt.Sprintf("http://appgate.com:%d", registry.Port),
					Version: 16,
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

			out := &bytes.Buffer{}
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(out)
			cmd.SetErr(io.Discard)

			stubber, teardown := prompt.InitAskStubber(t)
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
			if tt.wantOut != nil {
				got, err := io.ReadAll(out)
				if err != nil {
					t.Fatal("Test error: Failed to read output buffer")
				}
				if !tt.wantOut.Match(got) {
					t.Fatalf("WANT: %s\nGOT: %s", tt.wantOut.String(), string(got))
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

func Test_showPrepareUpgradeMessage(t *testing.T) {
	type args struct {
		f                             string
		appliance                     []openapi.Appliance
		skip                          []appliancepkg.SkipUpgrade
		stats                         []openapi.StatsAppliancesListAllOfData
		multiControllerUpgradeWarning bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "prepare appliance default",
			args: args{
				f: "appgate-6.0.0-29426-release.img.zip",
				appliance: []openapi.Appliance{
					{
						Id:   openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name: "controller1",
					},
					{
						Id:   openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name: "controller2",
					},
					{
						Id:   openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name: "gateway",
					},
				},
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Id:      openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name:    openapi.PtrString("controller1"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name:    openapi.PtrString("controller2"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name:    openapi.PtrString("gateway"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("92a8ceed-a364-4e99-a2eb-0a8546bab48f"),
						Name:    openapi.PtrString("controller3"),
						Online:  openapi.PtrBool(false),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("57a06ae4-8204-4780-a7c2-a9cdf03e5a0f"),
						Name:    openapi.PtrString("gateway2"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("6.0.0+29426"),
					},
				},
				skip: []appliancepkg.SkipUpgrade{
					{
						Appliance: openapi.Appliance{
							Id:   openapi.PtrString("92a8ceed-a364-4e99-a2eb-0a8546bab48f"),
							Name: "controller3",
						},
						Reason: "appliance is offline",
					},
					{
						Appliance: openapi.Appliance{
							Id:   openapi.PtrString("57a06ae4-8204-4780-a7c2-a9cdf03e5a0f"),
							Name: "gateway2",
						},
						Reason: "version is already greater or equal to prepare version",
					},
				},
			},
			want: `PREPARE SUMMARY

1. Upload upgrade image appgate-6.0.0-29426-release.img.zip to Controller
2. Prepare upgrade on the following appliances:

Appliance      Online    Current version    Prepare version
---------      ------    ---------------    ---------------
controller1    ✓         5.5.7+28767        6.0.0+29426
controller2    ✓         5.5.7+28767        6.0.0+29426
gateway        ✓         5.5.7+28767        6.0.0+29426


The following appliances will be skipped:

Appliance      Online    Current version    Reason
---------      ------    ---------------    ------
controller3    ⨯         5.5.7+28767        appliance is offline
gateway2       ✓         6.0.0+29426        version is already greater or equal to prepare version

`,
		},
		{
			name: "prepare appliance no-skipped",
			args: args{
				f: "appgate-6.0.0-29426-release.img.zip",
				appliance: []openapi.Appliance{
					{
						Id:   openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name: "controller1",
					},
					{
						Id:   openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name: "controller2",
					},
					{
						Id:   openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name: "gateway",
					},
				},
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Id:      openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name:    openapi.PtrString("controller1"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name:    openapi.PtrString("controller2"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name:    openapi.PtrString("gateway"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
				},
			},
			want: `PREPARE SUMMARY

1. Upload upgrade image appgate-6.0.0-29426-release.img.zip to Controller
2. Prepare upgrade on the following appliances:

Appliance      Online    Current version    Prepare version
---------      ------    ---------------    ---------------
controller1    ✓         5.5.7+28767        6.0.0+29426
controller2    ✓         5.5.7+28767        6.0.0+29426
gateway        ✓         5.5.7+28767        6.0.0+29426

`,
		},
		{
			name: "prepare appliance no-skipped",
			args: args{
				f:                             "appgate-6.0.0-29426-release.img.zip",
				multiControllerUpgradeWarning: true,
				appliance: []openapi.Appliance{
					{
						Id:   openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name: "controller1",
					},
					{
						Id:   openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name: "controller2",
					},
					{
						Id:   openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name: "gateway",
					},
				},
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Id:      openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name:    openapi.PtrString("controller1"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name:    openapi.PtrString("controller2"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:      openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name:    openapi.PtrString("gateway"),
						Online:  openapi.PtrBool(true),
						Version: openapi.PtrString("5.5.7+28767"),
					},
				},
			},
			want: `PREPARE SUMMARY

1. Upload upgrade image appgate-6.0.0-29426-release.img.zip to Controller
2. Prepare upgrade on the following appliances:

Appliance      Online    Current version    Prepare version
---------      ------    ---------------    ---------------
controller1    ✓         5.5.7+28767        6.0.0+29426
controller2    ✓         5.5.7+28767        6.0.0+29426
gateway        ✓         5.5.7+28767        6.0.0+29426


3. Delete upgrade image from Controller

WARNING: This upgrade requires all controllers to be upgraded to the same version, but not all
controllers are being prepared for upgrade.
A partial major or minor controller upgrade is not supported. The upgrade will fail unless all
controllers are prepared for upgrade when running 'upgrade complete'.
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareVersion, err := appliancepkg.ParseVersionString("6.0.0-29426-release.img.zip")
			if err != nil {
				t.Fatalf("internal test error: %v", err)
			}
			got, err := showPrepareUpgradeMessage(tt.args.f, prepareVersion, tt.args.appliance, tt.args.skip, tt.args.stats, tt.args.multiControllerUpgradeWarning)
			if (err != nil) != tt.wantErr {
				t.Errorf("showPrepareUpgradeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
