package appliance

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

var inactiveApplianceListResponse = `{
    "data": [
        {
            "id": "08cd20c0-f175-4503-96f7-c5b429c19236",
            "name": "new gateway",
            "notes": "",
            "created": "2021-11-02T14:11:50.122299Z",
            "updated": "2021-11-02T14:13:33.591830Z",
            "tags": [],
            "activated": false,
            "pendingCertificateRenewal": false,
            "version": 14,
            "hostname": "beta.devops",
            "site": "8a4add9e-0e99-4bb1-949c-c9faf9a49ad4",
            "siteName": "Default Site",
            "connectToPeersUsingClientPortWithSpa": true,
            "clientInterface": {
                "proxyProtocol": false,
                "hostname": "beta.devops",
                "httpsPort": 443,
                "dtlsPort": 443,
                "allowSources": [
                    {
                        "address": "0.0.0.0",
                        "netmask": 0
                    },
                    {
                        "address": "::",
                        "netmask": 0
                    }
                ]
            },
            "peerInterface": {
                "hostname": "beta.devops",
                "httpsPort": 444,
                "allowSources": [
                    {
                        "address": "0.0.0.0",
                        "netmask": 0
                    },
                    {
                        "address": "::",
                        "netmask": 0
                    }
                ]
            },
            "networking": {
                "hosts": [],
                "nics": [
                    {
                        "enabled": true,
                        "name": "eth0",
                        "ipv4": {
                            "dhcp": {
                                "enabled": false,
                                "dns": true,
                                "routers": true,
                                "ntp": false,
                                "mtu": false
                            },
                            "static": [
                                {
                                    "address": "10.97.158.3",
                                    "netmask": 26,
                                    "snat": false
                                }
                            ]
                        },
                        "ipv6": {
                            "dhcp": {
                                "enabled": false,
                                "dns": true,
                                "routers": false,
                                "ntp": false,
                                "mtu": false
                            },
                            "static": []
                        }
                    },
                    {
                        "enabled": true,
                        "name": "eth1",
                        "ipv4": {
                            "dhcp": {
                                "enabled": false,
                                "dns": true,
                                "routers": true,
                                "ntp": false,
                                "mtu": false
                            },
                            "static": [
                                {
                                    "address": "10.97.219.66",
                                    "netmask": 26,
                                    "snat": false
                                }
                            ]
                        },
                        "ipv6": {
                            "dhcp": {
                                "enabled": false,
                                "dns": true,
                                "routers": false,
                                "ntp": false,
                                "mtu": false
                            },
                            "static": []
                        }
                    }
                ],
                "dnsServers": [
                    "1.1.1.1",
                    "8.8.8.8"
                ],
                "dnsDomains": [],
                "routes": [
                    {
                        "address": "0.0.0.0",
                        "netmask": 0,
                        "gateway": "10.97.158.1"
                    }
                ]
            },
            "ntp": {
                "servers": [
                    {
                        "hostname": "0.ubuntu.pool.ntp.org"
                    },
                    {
                        "hostname": "1.ubuntu.pool.ntp.org"
                    },
                    {
                        "hostname": "2.ubuntu.pool.ntp.org"
                    },
                    {
                        "hostname": "3.ubuntu.pool.ntp.org"
                    }
                ]
            },
            "sshServer": {
                "enabled": true,
                "port": 22,
                "allowSources": [
                    {
                        "address": "0.0.0.0",
                        "netmask": 0
                    },
                    {
                        "address": "::",
                        "netmask": 0
                    }
                ],
                "passwordAuthentication": true
            },
            "snmpServer": {
                "enabled": false,
                "allowSources": []
            },
            "healthcheckServer": {
                "enabled": false,
                "port": 5555,
                "allowSources": [
                    {
                        "address": "0.0.0.0",
                        "netmask": 0
                    },
                    {
                        "address": "::",
                        "netmask": 0
                    }
                ]
            },
            "prometheusExporter": {
                "enabled": false,
                "port": 5556,
                "allowSources": []
            },
            "ping": {
                "allowSources": [
                    {
                        "address": "0.0.0.0",
                        "netmask": 0
                    },
                    {
                        "address": "::",
                        "netmask": 0
                    }
                ]
            },
            "logServer": {
                "enabled": false,
                "retentionDays": 30
            },
            "controller": {
                "enabled": false,
                "database": {
                    "location": "internal"
                }
            },
            "gateway": {
                "enabled": true,
                "vpn": {
                    "weight": 100,
                    "allowDestinations": [
                        {
                            "nic": "eth1"
                        }
                    ]
                }
            },
            "logForwarder": {
                "enabled": false,
                "tcpClients": [],
                "awsKineses": [],
                "sites": []
            },
            "connector": {
                "enabled": false,
                "expressClients": [],
                "advancedClients": []
            },
            "rsyslogDestinations": [],
            "hostnameAliases": []
        }
    ],
    "range": "0-2/2",
    "orderBy": "name",
    "descending": false,
    "filterBy": []
}`

func TestNewSeedCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		askStubs   func(*prompt.AskStubber)
		httpStubs  []httpmock.Stub
		wantErr    bool
		wantErrOut *regexp.Regexp
		wantJSON   bool
	}{
		{
			name: "json seed cloud",
			args: []string{
				"08cd20c0-f175-4503-96f7-c5b429c19236",
				"--allow-customization",
				"--provide-cloud-ssh-key",
				"--json",
			},
			wantJSON: true,
			httpStubs: []httpmock.Stub{
				{
					URL: "/appliances/08cd20c0-f175-4503-96f7-c5b429c19236",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
		                        "id": "08cd20c0-f175-4503-96f7-c5b429c19236",
		                        "name": "new gateway"
		                    }`))
						}
					},
				},
				{
					URL: "/appliances/08cd20c0-f175-4503-96f7-c5b429c19236/export",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost {
							panic("test error: expected only HTTP POST")
						}
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{
		                    "seed_data": "data"
		                }`))
					},
				},
			},
		},
		{
			name: "seed interactive",
			askStubs: func(s *prompt.AskStubber) {
				s.StubPrompt("select appliance:").AnswerDefault()
				s.StubPrompt("Seed type:").AnswerWith("ISO format")
				s.StubPrompt("SSH Authentication Method:").AnswerWith("Use SSH key provided by the cloud instance")
			},
			httpStubs: []httpmock.Stub{
				{
					URL: "/appliances",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(inactiveApplianceListResponse))
					},
				},

				{
					URL: "/appliances/08cd20c0-f175-4503-96f7-c5b429c19236",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
				                "id": "08cd20c0-f175-4503-96f7-c5b429c19236",
				                "name": "new gateway"
				            }`))
						}
					},
				},
				{
					URL: "/appliances/08cd20c0-f175-4503-96f7-c5b429c19236/export/iso",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost {
							panic("test error: expected only HTTP POST")
						}
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{
				            "seed_data": "data"
				        }`))
					},
				},
			},
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
					URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
				},
				IOOutWriter: stdout,
				Stdin:       pty,
				StdErr:      pty,
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registry.Client, nil
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
			cmd := NewSeedCmd(f)
			cmd.PersistentFlags().Bool("no-interactive", false, "suppress interactive prompt with auto accept")
			cmd.SetArgs(tt.args)

			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSeedCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", tt.wantErrOut, err.Error())
				}
				return
			}
			got, err := io.ReadAll(stdout)
			if err != nil {
				t.Fatalf("unable to read stdout %s", err)
			}
			if tt.wantJSON {
				if !util.IsJSON(string(got)) {
					t.Fatalf("Expected JSON output - got stdout\n%q\n", string(got))
				}
			}
			filename := "08cd20c0-f175-4503-96f7-c5b429c19236_seed.iso"
			if v, err := util.FileExists(filename); v && err == nil {
				os.Remove(filename)
			}

		})
	}
}
