package appliance

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

func TestSwitchPartition(t *testing.T) {
	// mutatingFunc := func(count int, b []byte) ([]byte, error) {
	// 	stats := &openapi.StatsAppliancesList{}
	// 	if err := json.Unmarshal(b, stats); err != nil {
	// 		return nil, err
	// 	}
	// 	data := stats.GetData()
	// 	for i := 0; i < len(data); i++ {
	// 		data[i].VolumeNumber = openapi.PtrFloat32(float32(count))
	// 	}
	// 	bytes, err := json.Marshal(stats)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return bytes, nil
	// }

	testCases := []struct {
		desc     string
		args     []string
		tty      bool
		apiStubs []httpmock.Stub
		askStubs func(*prompt.AskStubber)
		wantErr  bool
		expect   *regexp.Regexp
	}{
		// {
		// 	desc: "no arg",
		// 	tty:  true,
		// 	askStubs: func(s *prompt.AskStubber) {
		// 		s.StubPrompt("select appliance:").AnswerWith("controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1 - Default Site - []")
		// 		s.StubOne(true) // Confirmation prompt
		// 	},
		// 	apiStubs: []httpmock.Stub{
		// 		{
		// 			URL:       "/appliances",
		// 			Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
		// 		},
		// 		{
		// 			URL:       "/stats/appliances",
		// 			Responder: httpmock.MutatingResponse("../../pkg/appliance/fixtures/stats_appliance_6.2.6.json", mutatingFunc),
		// 		},
		// 		{
		// 			URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc",
		// 			Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_single.json"),
		// 		},
		// 		{
		// 			URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/switch-partition",
		// 			Responder: func(w http.ResponseWriter, r *http.Request) {
		// 				w.WriteHeader(http.StatusAccepted)
		// 			},
		// 		},
		// 	},
		// 	expect: regexp.MustCompile(`switched partition on controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1`),
		// },
		// {
		// 	desc: "with id arg",
		// 	tty:  true,
		// 	args: []string{"4c07bc67-57ea-42dd-b702-c2d6c45419fc"},
		// 	apiStubs: []httpmock.Stub{
		// 		{
		// 			URL:       "/stats/appliances",
		// 			Responder: httpmock.MutatingResponse("../../pkg/appliance/fixtures/stats_appliance_6.2.6.json", mutatingFunc),
		// 		},
		// 		{
		// 			URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc",
		// 			Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_single.json"),
		// 		},
		// 		{
		// 			URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/switch-partition",
		// 			Responder: func(w http.ResponseWriter, r *http.Request) {
		// 				w.WriteHeader(http.StatusAccepted)
		// 			},
		// 		},
		// 	},
		// 	askStubs: func(as *prompt.AskStubber) {
		// 		as.StubOne(true) // Confirmation prompt
		// 	},
		// 	expect: regexp.MustCompile(`switched partition on controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1`),
		// },
		{
			desc:    "with invalid arg",
			args:    []string{"dslkjflkjdsaf"},
			tty:     true,
			wantErr: true,
			expect:  regexp.MustCompile(`'dslkjflkjdsaf' is not a valid appliance ID`),
		},
		{
			desc: "no selection",
			tty:  true,
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
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance_6.2.6.json"),
				},
			},
			wantErr: true,
			expect:  regexp.MustCompile(`Answer "" not found in options`),
		},
		{
			desc:    "no TTY, no argument",
			wantErr: true,
			expect:  regexp.MustCompile(`no TTY present and no appliance ID provided`),
		},
		// {
		// 	desc: "no TTY, with argument",
		// 	args: []string{"4c07bc67-57ea-42dd-b702-c2d6c45419fc"},
		// 	apiStubs: []httpmock.Stub{
		// 		{
		// 			URL:       "/stats/appliances",
		// 			Responder: httpmock.MutatingResponse("../../pkg/appliance/fixtures/stats_appliance_6.2.6.json", mutatingFunc),
		// 		},
		// 		{
		// 			URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc",
		// 			Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_single.json"),
		// 		},
		// 		{
		// 			URL: "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/switch-partition",
		// 			Responder: func(w http.ResponseWriter, r *http.Request) {
		// 				w.WriteHeader(http.StatusAccepted)
		// 			},
		// 		},
		// 	},
		// 	expect: regexp.MustCompile(`switched partition on controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1`),
		// },
		// {
		// 	desc: "no user confirmation",
		// 	args: []string{"4c07bc67-57ea-42dd-b702-c2d6c45419fc"},
		// 	tty:  true,
		// 	apiStubs: []httpmock.Stub{
		// 		{
		// 			URL:       "/stats/appliances",
		// 			Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance_6.2.6.json"),
		// 		},
		// 		{
		// 			URL:       "/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc",
		// 			Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_single.json"),
		// 		},
		// 	},
		// 	askStubs: func(as *prompt.AskStubber) {
		// 		as.StubOne(false) // User confirmation
		// 	},
		// 	wantErr: true,
		// 	expect:  regexp.MustCompile(cmdutil.ErrExecutionCanceledByUser.Error()),
		// },
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

			stdin := &bytes.Buffer{}
			in := io.NopCloser(stdin)
			stderr := &bytes.Buffer{}
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
			if !tt.tty {
				f.Stdin = in
				f.StdErr = stderr
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
