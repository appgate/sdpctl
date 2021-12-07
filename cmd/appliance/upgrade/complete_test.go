package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/appgate/appgatectl/pkg/appliance"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/shlex"
)

type mockApplianceStatus struct{}

func (u *mockApplianceStatus) WaitForState(ctx context.Context, appliances []openapi.Appliance, expectedState string) error {
	return nil
}

type errApplianceStatus struct{}

func (u *errApplianceStatus) WaitForState(ctx context.Context, appliances []openapi.Appliance, expectedState string) error {
	return fmt.Errorf("never reached expected state %s", expectedState)
}

func TestUpgradeCompleteCommand(t *testing.T) {
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
			name: "first controller failed",
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
			wantErrOut:          regexp.MustCompile(`gateway never reached idle, got failed`),
			wantErr:             true,
		},
		{
			name: "one offline controller",
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

			registery := httpmock.NewRegistry()
			for _, v := range tt.httpStubs {
				registery.Register(v.URL, v.Responder)
			}

			defer registery.Teardown()
			registery.Serve()
			stdout := &bytes.Buffer{}
			stdin := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			in := io.NopCloser(stdin)
			f := &factory.Factory{
				Config: &configuration.Config{
					Debug:                    false,
					URL:                      fmt.Sprintf("http://localhost:%d", registery.Port),
					PrimaryControllerVersion: "5.3.4-24950",
				},
				IOOutWriter: stdout,
				Stdin:       in,
				StdErr:      stderr,
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registery.Client, nil
			}
			f.Appliance = func(c *configuration.Config) (*appliance.Appliance, error) {
				api, _ := f.APIClient(c)

				a := &appliance.Appliance{
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
			cmd := NewUpgradeCompleteCmd(f)
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
				t.Fatalf("TestUpgradeCompleteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", tt.wantErrOut, err.Error())
				}
			}
		})
	}
}
