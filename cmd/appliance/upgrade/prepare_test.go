package upgrade

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/appgate/appgatectl/pkg/appliance"
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

func TestUpgradePrepareCommand(t *testing.T) {

	tests := []struct {
		name       string
		cli        string
		askStubs   func(*prompt.AskStubber)
		httpStubs  []httpmock.Stub
		wantErr    bool
		wantErrOut *regexp.Regexp
	}{
		{
			name: "with existing file",
			cli:  "prepare --image './testdata/img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true) // peer_warning message
				s.StubOne(true) // backup confirmation
				s.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_idle.json"),
				},
				{
					URL:       "/files/img.zip",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL:       "/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
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
				s.StubOne(false) // peer_warning message
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Cancelled by user`),
		},
		{
			name: "no backup confirmation",
			cli:  "prepare --image './testdata/img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true)  // peer_warning message
				s.StubOne(false) // backup confirmation
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
			},
			wantErr: true,
		},
		{
			name: "no prepare confirmation",
			cli:  "prepare --image './testdata/img.zip'",
			askStubs: func(s *prompt.AskStubber) {
				s.StubOne(true)  // peer_warning message
				s.StubOne(true)  // backup confirmation
				s.StubOne(false) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
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
					Responder: httpmock.FileResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Image file not found "abc123456"`),
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
					Debug: false,
					URL:   fmt.Sprintf("http://localhost:%d", registery.Port),
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
				t.Fatalf("TestUpgradePrepareCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Logf("Stdout: %s", stdout)
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", tt.wantErrOut, err.Error())
				}
			}
		})
	}
}
