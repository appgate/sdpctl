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

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/google/shlex"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
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
					URL:   fmt.Sprintf("http://appgate.com:%d", registry.Port),
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

func TestPrintCompleteSummary(t *testing.T) {
	tests := []struct {
		name                  string
		primaryController     *openapi.Appliance
		additionalControllers []openapi.Appliance
		logServersForwarders  []openapi.Appliance
		chunks                [][]openapi.Appliance
		skipped               []openapi.Appliance
		backup                []openapi.Appliance
		backupDestination     string
		toVersion             string
		expect                string
	}{
		{
			name: "all upgrade no skip",
			primaryController: &openapi.Appliance{
				Name: "primary-controller",
			},
			additionalControllers: []openapi.Appliance{
				{Name: "secondary-controller"},
			},
			logServersForwarders: []openapi.Appliance{},
			chunks: [][]openapi.Appliance{
				{
					{Name: "gateway"},
					{Name: "gateway-2"},
				},
			},
			skipped:   []openapi.Appliance{},
			toVersion: "5.5.4",
			expect: `
UPGRADE COMPLETE SUMMARY

Appliances will be upgraded to version 5.5.4

Upgrade will be completed in steps:

 1. The primary Controller will be upgraded.
    This will result in the API being unreachable while completing the primary Controller upgrade.

    - primary-controller


 2. Additional Controllers will be upgraded.
    In some cases, the Controller function on additional Controllers will need to be disabled
    before proceeding with the upgrade. The disabled Controllers will then be re-enabled once
    the upgrade is completed.
    This step will also reboot the upgraded Controllers for the upgrade to take effect.

    - secondary-controller


 3. Additional appliances will be upgraded. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process.
    Some of the additional appliances may need to be rebooted for the upgrade to take effect.

    Batch #1:
    - gateway
    - gateway-2

`,
		},
		{
			name: "two upgrade two skipped",
			primaryController: &openapi.Appliance{
				Name: "primary-controller",
			},
			additionalControllers: []openapi.Appliance{},
			logServersForwarders:  []openapi.Appliance{},
			chunks: [][]openapi.Appliance{
				{
					{
						Name: "gateway",
					},
					{
						Name: "gateway-2",
					},
				},
			},
			skipped: []openapi.Appliance{
				{
					Name: "secondary-controller",
				},
				{
					Name: "additional-controller",
				},
			},
			backup: []openapi.Appliance{
				{
					Name: "primary-controller",
				},
			},
			backupDestination: "/tmp/appgate/backup",
			toVersion:         "5.5.4",
			expect: `
UPGRADE COMPLETE SUMMARY

Appliances will be upgraded to version 5.5.4

Upgrade will be completed in steps:

 1. Backup will be performed on the selected appliances
    and downloaded to /tmp/appgate/backup:

    - primary-controller


 2. The primary Controller will be upgraded.
    This will result in the API being unreachable while completing the primary Controller upgrade.

    - primary-controller


 3. Additional appliances will be upgraded. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process.
    Some of the additional appliances may need to be rebooted for the upgrade to take effect.

    Batch #1:
    - gateway
    - gateway-2


Appliances that will be skipped:
  - secondary-controller
  - additional-controller
`,
		},
		{
			name: "with logserver and forwarders",
			primaryController: &openapi.Appliance{
				Name: "primary-controller",
			},
			additionalControllers: []openapi.Appliance{},
			logServersForwarders: []openapi.Appliance{
				{Name: "logforwarder1"},
				{Name: "logforwarder2"},
			},
			chunks: [][]openapi.Appliance{
				{
					{
						Name: "gateway",
					},
					{
						Name: "gateway-2",
					},
				},
			},
			skipped: []openapi.Appliance{
				{
					Name: "secondary-controller",
				},
				{
					Name: "additional-controller",
				},
			},
			backup: []openapi.Appliance{
				{
					Name: "primary-controller",
				},
			},
			backupDestination: "/tmp/appgate/backup",
			toVersion:         "5.5.4",
			expect: `
UPGRADE COMPLETE SUMMARY

Appliances will be upgraded to version 5.5.4

Upgrade will be completed in steps:

 1. Backup will be performed on the selected appliances
    and downloaded to /tmp/appgate/backup:

    - primary-controller


 2. The primary Controller will be upgraded.
    This will result in the API being unreachable while completing the primary Controller upgrade.

    - primary-controller


 3. Appliances with LogForwarder/LogServer functions are updated
    Other appliances need a connection to to these appliances for logging.

    - logforwarder1
    - logforwarder2


 4. Additional appliances will be upgraded. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process.
    Some of the additional appliances may need to be rebooted for the upgrade to take effect.

    Batch #1:
    - gateway
    - gateway-2


Appliances that will be skipped:
  - secondary-controller
  - additional-controller
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			version, err := version.NewVersion(tt.toVersion)
			if err != nil {
				t.Fatalf("Failed to parse toVersion %s", err)
			}
			res, err := printCompleteSummary(&b, tt.primaryController, tt.additionalControllers, tt.logServersForwarders, tt.chunks, tt.skipped, tt.backup, tt.backupDestination, version)
			if err != nil {
				t.Errorf("printCompleteSummary() error - %s", err)
			}
			assert.Equal(t, tt.expect, res)
		})
	}
}

func TestPrintPostCompleteSummary(t *testing.T) {
	testCases := []struct {
		name              string
		applianceVersions map[string]string
		hasDiff           bool
		expect            string
	}{
		{
			name: "print no diff summary",
			applianceVersions: map[string]string{
				"controller": "6.0.0+12345",
				"gateway":    "6.0.0+12345",
			},
			hasDiff: false,
			expect: `UPGRADE COMPLETE

Appliance     Upgraded to
---------     -----------
controller    6.0.0+12345
gateway       6.0.0+12345

`,
		},
		{
			name: "diff on three appliances",
			applianceVersions: map[string]string{
				"primary-controller":   "6.0.0-beta+12345",
				"secondary-controller": "6.0.0+23456",
				"gateway":              "6.0.0+23456",
			},
			hasDiff: true,
			expect: `UPGRADE COMPLETE

Appliance               Upgraded to
---------               -----------
gateway                 6.0.0+23456
primary-controller      6.0.0-beta+12345
secondary-controller    6.0.0+23456

WARNING: Upgrade was completed, but not all appliances are running the same version.
`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := printPostCompleteSummary(tt.applianceVersions, tt.hasDiff)
			if err != nil {
				t.Fatal("error printing summary")
			}
			assert.Equal(t, tt.expect, res)
		})
	}
}
