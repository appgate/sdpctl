package appliance

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

func TestSwitchPartition(t *testing.T) {
	testCases := []struct {
		desc     string
		args     []string
		apiStubs []httpmock.Stub
		askStubs func(*prompt.AskStubber)
		wantErr  bool
		expect   *regexp.Regexp
	}{
		{
			desc: "no arg",
			askStubs: func(s *prompt.AskStubber) {
				s.StubPrompt("select appliance:").AnswerWith("controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1 - Default Site - []")
			},
			apiStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_single.json"),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/switch-partition",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusAccepted)
					},
				},
			},
			expect: regexp.MustCompile(`switched partition on controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1`),
		},
		{
			desc: "with id arg",
			args: []string{"4c07bc67-57ea-42dd-b702-c2d6c45419fc"},
			apiStubs: []httpmock.Stub{
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_single.json"),
				},
				{
					URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/switch-partition",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusAccepted)
					},
				},
			},
			expect: regexp.MustCompile(`switched partition on controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1`),
		},
		{
			desc:    "with invalid arg",
			args:    []string{"dslkjflkjdsaf"},
			wantErr: true,
			expect:  regexp.MustCompile(`'dslkjflkjdsaf' is not a valid appliance ID`),
		},
		{
			desc: "no selection",
			askStubs: func(s *prompt.AskStubber) {
				s.StubPrompt("select appliance:").AnswerWith("")
			},
			apiStubs: []httpmock.Stub{
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance.json"),
				},
			},
			wantErr: true,
			expect:  regexp.MustCompile(`Answer "" not found in options`),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			registry := httpmock.NewRegistry(t)
			for _, stub := range tt.apiStubs {
				registry.RegisterStub(stub)
			}
			defer registry.Teardown()
			registry.Serve()

			pty, tty, err := pseudotty.Open()
			if err != nil {
				t.Fatalf("failed to open pseudotty: %v", err)
			}
			term := vt10x.New(vt10x.WithWriter(tty))
			c, err := expect.NewConsole(expect.WithStdin(pty), expect.WithStdout(term), expect.WithCloser(pty, tty))
			if err != nil {
				t.Fatalf("failed to create console: %v", err)
			}
			defer c.Close()
			stdout := &bytes.Buffer{}

			url := fmt.Sprintf("http://localhost:%d", registry.Port)
			f := &factory.Factory{
				Config: &configuration.Config{
					Debug: false,
					URL:   url,
				},
				IOOutWriter: stdout,
				Stdin:       pty,
				StdErr:      pty,
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registry.Client, nil
			}
			f.BaseURL = func() string {
				return url + "/admin"
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

			stubber, teardown := prompt.InitAskStubber(t)
			defer teardown()
			if tt.askStubs != nil {
				tt.askStubs(stubber)
			}

			cmd := NewApplianceCmd(f)
			args := []string{"switch-partition"}
			args = append(args, tt.args...)
			cmd.SetArgs(args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("want: %v, got: %v", tt.wantErr, err)
			} else if err != nil && !tt.expect.MatchString(err.Error()) {
				t.Fatalf("unexpected error output: want - %v, got - %s", tt.expect.String(), err.Error())
			}
			got, ierr := io.ReadAll(stdout)
			if ierr != nil {
				t.Fatalf("error reading output: %v", ierr)
			}
			if err == nil && !tt.expect.MatchString(string(got)) {
				want := tt.expect.String()
				t.Fatalf("unexpected command output: want - %v, got - %s", want, got)
			}
		})
	}
}
