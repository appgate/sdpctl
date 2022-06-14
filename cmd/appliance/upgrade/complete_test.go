package upgrade

import (
	"bytes"
	"context"
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
	"github.com/google/shlex"
	"github.com/hashicorp/go-version"
)

type mockApplianceStatus struct{}

func (u *mockApplianceStatus) WaitForState(ctx context.Context, appliance openapi.Appliance, expectedState string, status chan<- string) error {
	return nil
}
func (u *mockApplianceStatus) WaitForStatus(ctx context.Context, appliance openapi.Appliance, want []string) error {
	return nil
}

type errApplianceStatus struct{}

func (u *errApplianceStatus) WaitForState(ctx context.Context, appliance openapi.Appliance, expectedState string, status chan<- string) error {
	return fmt.Errorf("never reached expected state %s", expectedState)
}
func (u *errApplianceStatus) WaitForStatus(ctx context.Context, appliance openapi.Appliance, want []string) error {
	return fmt.Errorf("Never reached expected status %s", want)
}

func TestUpgradeCompleteCommand(t *testing.T) {
	applianceUUID := "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
	backupUUID := "fd5ea380-496b-41eb-8bc8-2c84eb36b605"

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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
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
			name: "first controller failed",
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
			},
			upgradeApplianeStatusWorker: &errApplianceStatus{},
			wantErrOut:                  regexp.MustCompile(`primary controller never reached expected state single_controller_ready`),
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
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
			name: "one offline controller",
			cli:  "upgrade complete --backup=false --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/two_controller_one_offline.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_stats_offline_controller.json"),
				},
			},
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`Could not complete upgrade operation 1 error occurred`),
			wantErr:             true,
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
					Debug:                    false,
					URL:                      fmt.Sprintf("http://appgate.com:%d", registry.Port),
					PrimaryControllerVersion: "5.3.4+24950",
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
				{
					Name: "secondary-controller",
				},
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
			skipped:   []openapi.Appliance{},
			toVersion: "5.5.4",
			expect: `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in three steps:

 1. The primary controller will be upgraded.
    This will result in the API being unreachable while completing the primary controller upgrade.

 2. Additional controllers will be upgraded.
    In some cases, the controller function on additional controllers will need to be disabled
    before proceeding with the upgrade. The disabled controllers will then be re-enabled once
    the upgrade is completed.
    This step will also reboot the upgraded controllers for the upgrade to take effect.

 3. The remaining appliances will be upgraded. The additional appliances will be split into
    batches to keep the collective as available as possible during the upgrade process.
    Some of the additional appliances may need to be rebooted for the upgrade to take effect.

The following appliances will be upgraded to version 5.5.4:
  Primary Controller: primary-controller

  Additional Controllers:
  - secondary-controller

  Additional Appliances:
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

Upgrade will be completed in three steps:

 1. The primary controller will be upgraded.
    This will result in the API being unreachable while completing the primary controller upgrade.

 2. Additional controllers will be upgraded.
    In some cases, the controller function on additional controllers will need to be disabled
    before proceeding with the upgrade. The disabled controllers will then be re-enabled once
    the upgrade is completed.
    This step will also reboot the upgraded controllers for the upgrade to take effect.

 3. The remaining appliances will be upgraded. The additional appliances will be split into
    batches to keep the collective as available as possible during the upgrade process.
    Some of the additional appliances may need to be rebooted for the upgrade to take effect.

The following appliances will be upgraded to version 5.5.4:
  Primary Controller: primary-controller

  Additional Appliances:
    Batch #1:
    - gateway
    - gateway-2

Appliances that will be skipped:
  - secondary-controller
  - additional-controller

Appliances that will be backed up before completing upgrade:
  - primary-controller
Backup destination is: /tmp/appgate/backup
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			version, _ := version.NewVersion(tt.toVersion)
			res, err := printCompleteSummary(&b, tt.primaryController, tt.additionalControllers, tt.chunks, tt.skipped, tt.backup, tt.backupDestination, version)
			if err != nil {
				t.Errorf("printCompleteSummary() error - %s", err)
			}
			if res != tt.expect {
				t.Errorf("printCompleteSummary() fail\nEXPECT:%s\nGOT:%s", tt.expect, res)
			}
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


Appliances are now running these versions:
  controller: 6.0.0+12345
  gateway: 6.0.0+12345
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

WARNING: Upgrade was completed, but not all appliances are running the same version.
Appliances are now running these versions:
  gateway: 6.0.0+23456
  primary-controller: 6.0.0-beta+12345
  secondary-controller: 6.0.0+23456
`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := printPostCompleteSummary(tt.applianceVersions, tt.hasDiff)
			if err != nil {
				t.Fatal("error printing summary")
			}
			if res != tt.expect {
				t.Fatalf("Output don't match expected:\nWANT: %s\nGOT: %s", tt.expect, res)
			}
		})
	}
}
