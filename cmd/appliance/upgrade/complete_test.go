package upgrade

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/google/shlex"
)

type mockApplianceStatus struct{}

func (u *mockApplianceStatus) WaitForState(timeout time.Duration, appliances []openapi.Appliance, expectedState string) error {
	return nil
}

type errApplianceStatus struct{}

func (u *errApplianceStatus) WaitForState(timeout time.Duration, appliances []openapi.Appliance, expectedState string) error {
	return fmt.Errorf("never reached expected state %s", expectedState)
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
			name: "test complete multiple appliances",
			cli:  "upgrade complete --backup=false",
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
			},
			wantErr: false,
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
			},
			wantErr: false,
		},
		{
			name: "test complete multiple appliances",
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
			},
			wantErr: false,
		},
		{
			name:       "upgrade workers error",
			cli:        "upgrade complete --throttle invalid",
			wantErr:    true,
			wantErrOut: regexp.MustCompile("invalid syntax"),
		},
		{
			name: "first controller failed",
			cli:  "upgrade complete --backup=false",
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
			},
			upgradeApplianeStatusWorker: &errApplianceStatus{},
			wantErrOut:                  regexp.MustCompile(`primary controller never reached expected state single_controller_ready`),
			wantErr:                     true,
		},
		{
			name: "gateway failure",
			cli:  "upgrade complete --backup=false",
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
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
			},
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`gateway never reached ready, got failed`),
			wantErr:             true,
		},
		{
			name: "one offline controller",
			cli:  "upgrade complete --backup=false",
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
