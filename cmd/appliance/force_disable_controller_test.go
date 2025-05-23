package appliance

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/dns"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	pseudotty "github.com/creack/pty"
	"github.com/foxcpp/go-mockdns"
	"github.com/google/shlex"
	"github.com/google/uuid"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
)

func TestForceDisableControllerCMD(t *testing.T) {
	tests := []struct {
		name       string
		cli        string
		httpStubs  []httpmock.Stub
		askStubs   func(*prompt.PromptStubber)
		wantErr    bool
		wantErrOut *regexp.Regexp
	}{
		{
			name:       "no arguments, no interactive",
			cli:        "--no-interactive",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`No arguments provided while running in no-interactive mode`),
		},
		{
			name: "disable one controller",
			cli:  "cryptzone.com --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_stats_appliance_one_offline.json"),
				},
				{
					URL: "/admin/appliances/force-disable-controllers",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "ba86a668-a965-44bb-a6b0-07df8f449c01")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/21ac20ec-410a-4b59-baf3-fdacbe455581/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/repartition-ip-allocations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "1012ad03-f4ac-4760-ab21-b9bfc2c769d7")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/21ac20ec-410a-4b59-baf3-fdacbe455581/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
			},
		},
		{
			name: "disable two controller",
			cli:  "cryptzone.com ctrl3.cryptzone.com --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_stats_appliance_one_offline.json"),
				},
				{
					URL: "/admin/appliances/force-disable-controllers",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "ba86a668-a965-44bb-a6b0-07df8f449c01")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/repartition-ip-allocations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "1012ad03-f4ac-4760-ab21-b9bfc2c769d7")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
			},
		},
		{
			name: "disable two controllers using ID",
			cli:  "ed95fac8-9098-472b-b9f0-fe741881e2ca ctrl3.cryptzone.com --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_stats_appliance_one_offline.json"),
				},
				{
					URL: "/admin/appliances/force-disable-controllers",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "ba86a668-a965-44bb-a6b0-07df8f449c01")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/repartition-ip-allocations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "1012ad03-f4ac-4760-ab21-b9bfc2c769d7")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
			},
		},
		{
			name: "disable offline controller",
			cli:  "ctrl4.cryptzone.com --no-interactive",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_stats_appliance_one_offline.json"),
				},
				{
					URL: "/admin/appliances/force-disable-controllers",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "ba86a668-a965-44bb-a6b0-07df8f449c01")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/ed95fac8-9098-472b-b9f0-fe741881e2ca/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/21ac20ec-410a-4b59-baf3-fdacbe455581/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/repartition-ip-allocations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "1012ad03-f4ac-4760-ab21-b9bfc2c769d7")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/ed95fac8-9098-472b-b9f0-fe741881e2ca/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/21ac20ec-410a-4b59-baf3-fdacbe455581/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
			},
		},
		{
			name: "disable when name and hostnames are similar",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list_similar.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance_similar.json"),
				},
				{
					URL: "/admin/appliances/force-disable-controllers",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "ba86a668-a965-44bb-a6b0-07df8f449c01")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/ba86a668-a965-44bb-a6b0-07df8f449c01",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "ba86a668-a965-44bb-a6b0-07df8f449c01", "result": "success", "status": "completed", "details": ""}`))
					},
				},
				{
					URL: "/admin/appliances/repartition-ip-allocations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Change-ID", "1012ad03-f4ac-4760-ab21-b9bfc2c769d7")
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/1012ad03-f4ac-4760-ab21-b9bfc2c769d7",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.Write([]byte(`{"id": "1012ad03-f4ac-4760-ab21-b9bfc2c769d7", "result": "success", "status": "completed", "details": ""}`))
					},
				},
			},
			askStubs: func(as *prompt.PromptStubber) {
				as.StubPrompt("Select Controllers to force disable").AnswerWith([]string{"controller-site1 DR (ctrl.appgate.test)"})
				as.StubOne(true)
			},
		},
		{
			name:       "disable non existing hostname",
			cli:        "foobar --no-interactive",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`No Controllers selected to disable`),
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../pkg/appliance/fixtures/ha_stats_appliance_one_offline.json"),
				},
			},
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
			f := &factory.Factory{
				Config: &configuration.Config{
					Debug: false,
					URL:   fmt.Sprintf("https://appgate.test:%d", registry.Port),
				},
				IOOutWriter: stdout,
				Stdin:       pty,
				StdErr:      pty,
				APIClient:   func(c *configuration.Config) (*openapi.APIClient, error) { return registry.Client, nil },
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

			cmd := NewForceDisableControllerCmd(f)
			flags := cmd.PersistentFlags()
			flags.Bool("ci-mode", false, "ci-mode")
			flags.Bool("no-interactive", false, "no-interactive")
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
				t.Fatalf("TestForceDisableControllerCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("Expected output to match, expected:\n%s\n got: \n%s\n", tt.wantErrOut, err.Error())
				}
			}
		})
	}
}

func Test_printSummary(t *testing.T) {
	app1, app1data := generateApplianceWithStats("appliance1", "appliance1.example.com", "6.1.1-12345", "healthy")
	app2, app2data := generateApplianceWithStats("appliance2", "appliance2.example.com", "6.1.1-12345", "healthy")
	app3, app3data := generateApplianceWithStats("appliance3", "appliance3.example.com", "unknown", "offline")
	stats := openapi.NewApplianceWithStatusList()
	stats.Data = append(stats.Data, app1data, app2data, app3data)
	type args struct {
		stats   []openapi.ApplianceWithStatus
		disable []openapi.Appliance
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "disable one controller",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app1},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance1    appliance1.example.com    healthy    6.1.1-12345

`,
		},
		{
			name: "disable two controllers",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app1, app2},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance1    appliance1.example.com    healthy    6.1.1-12345
appliance2    appliance2.example.com    healthy    6.1.1-12345

`,
		},
		{
			name: "disable two controllers, one offline",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app1, app2},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance1    appliance1.example.com    healthy    6.1.1-12345
appliance2    appliance2.example.com    healthy    6.1.1-12345

`,
		},
		{
			name: "disable offline controller",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app3},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance3    appliance3.example.com    offline    unknown

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := printSummary(tt.args.stats, tt.args.disable)
			if (err != nil) != tt.wantErr {
				t.Errorf("printSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func generateApplianceWithStats(name, hostname, version, status string) (openapi.Appliance, openapi.ApplianceWithStatus) {
	app := openapi.NewApplianceWithDefaults()
	id := uuid.NewString()
	app.SetId(id)
	app.SetName(name)
	app.SetHostname(hostname)
	appstatdata := *openapi.NewApplianceWithStatusWithDefaults()
	appstatdata.SetId(app.GetId())
	appstatdata.SetStatus(status)
	appstatdata.SetApplianceVersion(version)
	return *app, appstatdata
}
