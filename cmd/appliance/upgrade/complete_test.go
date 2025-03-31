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

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
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
	tests := []struct {
		name                        string
		cli                         string
		appliances                  []string
		askStubs                    func(*prompt.PromptStubber)
		upgradeStatusWorker         appliancepkg.WaitForUpgradeStatus
		upgradeApplianeStatusWorker appliancepkg.WaitForApplianceStatus
		from, to                    string
		customStubs                 []httpmock.Stub
		wantErr                     bool
		wantErrOut                  *regexp.Regexp
	}{
		{
			name:       "with invalid arg",
			cli:        "upgrade complete some.invalid.arg",
			from:       "6.2",
			to:         "6.3",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`accepts 0 arg\(s\), received 1`),
		},
		{
			name: "test complete multiple appliances backup false",
			cli:  "upgrade complete --backup=false",
			askStubs: func(as *prompt.PromptStubber) {
				as.StubOne(true)
			},
			from: "6.2.0",
			to:   "6.2.1",
			appliances: []string{
				"primary",
				"secondary",
				"gatewayA1",
			},
			wantErr: false,
		},
		{
			name:       "test no appliances ready",
			cli:        "upgrade complete --no-interactive",
			appliances: []string{appliancepkg.TestApplianceUnpreparedPrimary, appliancepkg.TestApplianceControllerNotPrepared},
			customStubs: []httpmock.Stub{
				{
					URL: "/admin/appliances/{appliance}/upgrade",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						us := openapi.NewStatsAppliancesListAllOfUpgradeWithDefaults()
						us.SetStatus(appliancepkg.UpgradeStatusIdle)
						body, err := us.MarshalJSON()
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.Header().Add("Content-Type", "application/json")
						w.Write(body)
					},
				},
			},
			from:       "6.2.0",
			to:         "6.2.1",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`No appliances are ready to upgrade. Please run 'upgrade prepare' before trying to complete an upgrade`),
		},
		{
			name:       "test complete with filter function gateway",
			cli:        "upgrade complete --backup=false --include function=gateway --no-interactive",
			appliances: []string{appliancepkg.TestAppliancePrimary, appliancepkg.TestApplianceGatewayA1, appliancepkg.TestApplianceGatewayA2},
			from:       "6.2.0",
			to:         "6.2.1",
			wantErr:    false,
		},
		{
			// TODO; fails to Windows rename issue. See https://github.com/appgate/sdpctl/pull/22#pullrequestreview-813268386
			name:       "test complete multiple appliances with backup",
			cli:        "upgrade complete --backup=true --no-interactive=true",
			appliances: []string{appliancepkg.TestAppliancePrimary, appliancepkg.TestApplianceSecondary, appliancepkg.TestApplianceGatewayA1, appliancepkg.TestApplianceGatewayA2},
			from:       "6.2.0",
			to:         "6.2.1",
			wantErr:    false,
		},
		{
			name:                        "first Controller failed",
			cli:                         "upgrade complete --backup=false --no-interactive",
			appliances:                  []string{appliancepkg.TestAppliancePrimary},
			from:                        "6.2.0",
			to:                          "6.2.1",
			upgradeApplianeStatusWorker: &errApplianceStatus{},
			wantErrOut:                  regexp.MustCompile(`the primary Controller never reached expected state`),
			wantErr:                     true,
		},
		{
			name:                "gateway failure",
			cli:                 "upgrade complete --backup=false --no-interactive",
			appliances:          []string{appliancepkg.TestAppliancePrimary, appliancepkg.TestApplianceGatewayA1},
			from:                "6.2.0",
			to:                  "6.2.1",
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`gateway never reached idle, got failed`),
			wantErr:             true,
		},
		{
			name:                "one offline Controller",
			cli:                 "upgrade complete --backup=false --no-interactive",
			appliances:          []string{appliancepkg.TestAppliancePrimary, appliancepkg.TestApplianceControllerOffline},
			from:                "6.2.0",
			to:                  "6.2.1",
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(fmt.Sprintf(`Cannot start the operation since a Controller "%s" is offline`, appliancepkg.TestApplianceControllerOffline)),
			wantErr:             true,
		},
		{
			name:       "no volume switch",
			cli:        "upgrade complete --backup=false --no-interactive",
			from:       "6.2.0",
			to:         "6.2.1",
			appliances: []string{appliancepkg.TestAppliancePrimary},
			customStubs: []httpmock.Stub{
				{
					URL: "/admin/appliances/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						b, err := json.Marshal(appliancepkg.InitialTestStats)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.Header().Add("Content-Type", "application/json")
						w.Write(b)
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile("never switched partition"),
		},
		{
			// Allow some controllers to be unprepared. Proceed only upgrade-completing the
			// prepared controllers
			name:       "controller major-minor guard unprepared controller",
			cli:        "upgrade complete --backup=false --no-interactive",
			appliances: []string{appliancepkg.TestAppliancePrimary, appliancepkg.TestApplianceControllerNotPrepared},
			from:       "6.2.0",
			to:         "6.3.0",
			wantErr:    false,
		},
		{
			name:       "controller major-minor guard mismatch version",
			cli:        "upgrade complete --backup=false --no-interactive",
			appliances: []string{appliancepkg.TestAppliancePrimary, appliancepkg.TestApplianceControllerMismatch},
			from:       "6.2.0",
			to:         "6.3.0",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(appliancepkg.ErrControllerVersionMismatch.Error()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, teardown := dns.RunMockDNSServer(map[string]mockdns.Zone{})
			defer teardown()
			registry := httpmock.NewRegistry(t)
			hostname := "appgate.test"

			testColl := appliancepkg.GenerateCollective(t, hostname, tt.from, tt.to, tt.appliances)

			testApps := []openapi.Appliance{}
			for _, a := range tt.appliances {
				app := testColl.Appliances[a]
				testApps = append(testApps, app)
			}
			stubs := testColl.GenerateStubs(testApps, *testColl.Stats, *testColl.UpgradedStats)
			for _, st := range tt.customStubs {
				for i, v := range stubs {
					if st.URL == v.URL {
						stubs[i] = st
					}
				}
			}

			for _, v := range stubs {
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
			cmd.PersistentFlags().BoolP("help", "x", false, "")
			cmd.PersistentFlags().Bool("ci-mode", false, "")
			cmd.PersistentFlags().Bool("no-interactive", false, "usage")
			cmd.PersistentFlags().String("actual-hostname", "", "")

			argv, err := shlex.Split(tt.cli)
			if err != nil {
				panic("Internal testing error, failed to split args")
			}
			cmd.SetArgs(argv)

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			stubber, teardown := prompt.InitStubbers(t)
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
