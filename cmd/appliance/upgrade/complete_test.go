package upgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/dns"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/foxcpp/go-mockdns"
	"github.com/google/shlex"
)

type mockApplianceStatus struct{}

func (u *mockApplianceStatus) WaitForApplianceState(ctx context.Context, appliance openapi.Appliance, want []string, tracker *tui.Tracker) error {
	return nil
}
func (u *mockApplianceStatus) WaitForApplianceStatus(ctx context.Context, appliance openapi.Appliance, want []string, tracker *tui.Tracker) error {
	return nil
}

type errApplianceStatus struct{}

func (u *errApplianceStatus) WaitForApplianceState(ctx context.Context, appliance openapi.Appliance, want []string, tracker *tui.Tracker) error {
	return fmt.Errorf("never reached expected state %s", want)
}
func (u *errApplianceStatus) WaitForApplianceStatus(ctx context.Context, appliance openapi.Appliance, want []string, tracker *tui.Tracker) error {
	return fmt.Errorf("Never reached expected status %s", want)
}

func TestUpgradeCompleteCommand(t *testing.T) {
	applianceUUID := "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
	backupUUID := "fd5ea380-496b-41eb-8bc8-2c84eb36b605"

	mutatingFunc := func(count int, b []byte) ([]byte, error) {
		stats := &openapi.StatsAppliancesList{}
		if err := json.Unmarshal(b, stats); err != nil {
			return nil, err
		}
		data := stats.GetData()
		for i := 0; i < len(data); i++ {
			data[i].VolumeNumber = openapi.PtrFloat32(float32(count))
		}
		bytes, err := json.Marshal(stats)
		if err != nil {
			return nil, err
		}
		return bytes, nil
	}

	tests := []struct {
		name                        string
		cli                         string
		httpStubs                   []httpmock.Stub
		askStubs                    func(*prompt.AskStubber)
		upgradeStatusWorker         appliancepkg.WaitForUpgradeStatus
		upgradeApplianeStatusWorker appliancepkg.WaitForApplianceStatus
		wantErr                     bool
		wantErrOut                  *regexp.Regexp
	}{
		{
			name:       "with invalid arg",
			cli:        "upgrade complete some.invalid.arg",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`accepts 0 arg\(s\), received 1`),
		},
		{
			name: "test complete multiple appliances backup false",
			cli:  "upgrade complete --backup=false",
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
					Responder: httpmock.MutatingResponse("../../../pkg/appliance/fixtures/stats_appliance.json", mutatingFunc),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test no appliances ready",
			cli:  "upgrade complete --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.MutatingResponse("../../../pkg/appliance/fixtures/stats_appliance.json", mutatingFunc),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_idle.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_idle.json"),
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`No appliances are ready to upgrade. Please run 'upgrade prepare' before trying to complete an upgrade`),
		},
		{
			name: "test complete with filter function gateway",
			cli:  "upgrade complete --backup=false --filter function=gateway --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.MutatingResponse("../../../pkg/appliance/fixtures/stats_appliance.json", mutatingFunc),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
			},
			wantErr: false,
		},
		{
			// TODO; fails to Windows rename issue. See https://github.com/appgate/sdpctl/pull/22#pullrequestreview-813268386
			name: "test complete multiple appliances with backup",
			cli:  "upgrade complete --backup=true --no-interactive=true",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.MutatingResponse("../../../pkg/appliance/fixtures/stats_appliance.json", mutatingFunc),
				},
				{
					URL:       "/global-settings",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_global_options.json"),
				},
				{
					URL:       fmt.Sprintf("/appliances/%s/backup", applianceUUID),
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_backup_initiated.json"),
				},
				{
					URL:       fmt.Sprintf("/appliances/%s/backup/%s/status", applianceUUID, backupUUID),
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_backup_status_done.json"),
				},
				{
					URL:       fmt.Sprintf("/appliances/%s/backup/%s", applianceUUID, backupUUID),
					Responder: httpmock.FileResponse(),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
				{
					URL: "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
			},
			wantErr: false,
		},
		{
			name: "first Controller failed",
			cli:  "upgrade complete --backup=false --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.MutatingResponse("../../../pkg/appliance/fixtures/stats_appliance.json", mutatingFunc),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
			},
			upgradeApplianeStatusWorker: &errApplianceStatus{},
			wantErrOut:                  regexp.MustCompile(`the primary Controller never reached expected state`),
			wantErr:                     true,
		},
		{
			name: "gateway failure",
			cli:  "upgrade complete --backup=false --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.MutatingResponse("../../../pkg/appliance/fixtures/stats_appliance.json", mutatingFunc),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_failed.json"),
				},
			},
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`gateway never reached idle, got failed`),
			wantErr:             true,
		},
		{
			name: "one offline Controller",
			cli:  "upgrade complete --backup=false --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/two_controller_one_offline.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.MutatingResponse("../../../pkg/appliance/fixtures/appliance_stats_offline_controller.json", mutatingFunc),
				},
			},
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`Could not complete the upgrade operation 1 error occurred`),
			wantErr:             true,
		},
		{
			name: "no volume switch",
			cli:  "upgrade complete --backup=false --no-interactive",
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/complete",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile("never switched partition"),
		},
		{
			name: "controller major-minor guard unprepared controller",
			cli:  "upgrade complete --backup=false --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/ha_appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/ha_stats_appliance.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/21ac20ec-410a-4b59-baf3-fdacbe455581/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/9c557978-1dcd-4b42-ad56-afb6abf1490c/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/ed95fac8-9098-472b-b9f0-fe741881e2ca/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_idle.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile("All Controllers need upgrading when doing major or minor version upgrade, but not all controllers are prepared for upgrade. Please prepare the remaining controllers before running 'upgrade complete' again."),
		},
		{
			name: "controller major-minor guard mismatch version",
			cli:  "upgrade complete --backup=false --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/ha_appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/ha_stats_appliance.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/21ac20ec-410a-4b59-baf3-fdacbe455581/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL:       "/appliances/9c557978-1dcd-4b42-ad56-afb6abf1490c/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
				{
					URL: "/appliances/ed95fac8-9098-472b-b9f0-fe741881e2ca/upgrade",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "ready", "details": "appgate-5.5.0.img.zip" }`))
					},
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_ready.json"),
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile("Version mismatch on prepared Controllers. Controllers need to be prepared with the same version when doing a major or minor version upgrade."),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, teardown := dns.RunMockDNSServer(map[string]mockdns.Zone{})
			defer teardown()
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
					URL:   fmt.Sprintf("http://appgate.test:%d", registry.Port),
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
				if tt.upgradeApplianeStatusWorker != nil {
					a.ApplianceStats = tt.upgradeApplianeStatusWorker
				} else {
					a.ApplianceStats = new(mockApplianceStatus)
				}
				return a, nil
			}
			// add parent command to allow us to include test with parent flags
			cmd := NewApplianceCmd(f)
			upgradeCmd := NewUpgradeCmd(f)
			upgradeCmd.AddCommand(NewUpgradeCompleteCmd(f))
			cmd.AddCommand(upgradeCmd)

			// cobra hack
			cmd.Flags().BoolP("help", "x", false, "")
			cmd.Flags().Bool("no-interactive", false, "usage")
			cmd.PersistentFlags().String("actual-hostname", "", "")

			argv, err := shlex.Split(tt.cli)
			if err != nil {
				panic("Internal testing error, failed to split args")
			}
			cmd.SetArgs(argv)

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			stubber, teardown := prompt.InitAskStubber(t)
			defer teardown()

			if tt.askStubs != nil {
				tt.askStubs(stubber)
			}
			_, err = cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("TestUpgradeCompleteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("Expected output to match, expected:\n%s\n got: \n%s\n", tt.wantErrOut, err.Error())
				}
			}
		})
	}
}
