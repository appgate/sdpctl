package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"

	"github.com/appgate/sdpctl/pkg/prompt"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/zalando/go-keyring"
)

var (
	authenticationResponse = httpmock.Stub{
		URL: "/authentication",
		Responder: func(rw http.ResponseWriter, r *http.Request) {
			if v, ok := r.Header["Accept"]; ok && v[0] == "application/vnd.appgate.peer-v5+json" {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNotAcceptable)
				fmt.Fprint(rw, string(`{
                    "id": "string",
                    "message": "string",
                    "minSupportedVersion": 7,
                    "maxSupportedVersion": 15
                  }`))
				return
			}

			if r.Method == http.MethodPost {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "providerName": "ldap",
                    "username": "bob",
                    "password": "alice",
                    "deviceId": "4c07bc67-57ea-42dd-b702-c2d6c45419fc",
                    "samlResponse": "string"
                }`))
			}
		}}
	unauthorizedResponse = httpmock.Stub{
		URL: "/authentication",
		Responder: func(w http.ResponseWriter, r *http.Request) {
			if v, ok := r.Header["Accept"]; ok && v[0] == "application/vnd.appgate.peer-v5+json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotAcceptable)
				fmt.Fprint(w, string(`{
                    "id": "string",
                    "message": "string",
                    "minSupportedVersion": 7,
                    "maxSupportedVersion": 15
                  }`))
				return
			}

			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, string(`{"id":"unauthorized","message":"Invalid username or password.","failureType":"Login"}`))
			}
		},
	}
	identityProviderNames = httpmock.Stub{
		URL: "/identity-providers/names",
		Responder: func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                        "data": [
                            {
                                "name": "local",
                                "displayName": "local",
                                "type": "Credentials"
                            }
                        ]
                    }`))
			}
		}}

	authorizationGET = httpmock.Stub{
		URL: "/authorization",
		Responder: func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "user": {
                      "name": "admin",
                      "needTwoFactorAuth": false,
                      "canAccessAuditLogs": true,
                      "privileges": [
                        {
                          "type": "All",
                          "target": "All",
                          "scope": {
                            "all": true,
                            "ids": [
                              "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
                            ],
                            "tags": [
                              "tag"
                            ]
                          },
                          "defaultTags": [
                            "api-created"
                          ]
                        }
                      ]
                    },
                    "token": "string",
                    "expires": "2019-08-24T14:15:22Z",
                    "messageOfTheDay": "Welcome to Appgate SDP."
                  }`))
			}
		}}
)

func TestSignInNoPromptOrEnv(t *testing.T) {
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug:                    false,
			URL:                      "http://localhost",
			PrimaryControllerVersion: "5.3.4-24950",
		},
		IOOutWriter: os.Stdout,
		Stdin:       os.Stdin,
		StdErr:      os.Stderr,
	}
	err := Signin(f)
	if err == nil {
		t.Fatal("Expected err ErrSignInNotSupported")
	}
	if !errors.Is(ErrSignInNotSupported, err) {
		t.Errorf("expected %s, got %s", ErrSignInNotSupported, err)
	}
}

func TestSignin(t *testing.T) {

	tests := []struct {
		name                 string
		askStubs             func(*prompt.AskStubber)
		environmentVariables map[string]string
		httpStubs            []httpmock.Stub
		wantErr              bool
	}{
		{
			name: "signin with environment variables",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME": "bob",
				"SDPCTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderNames,
				authorizationGET,
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../appliance/fixtures/stats_appliance.json"),
				},
			},
		},
		{
			name: "signin prompt username and password",

			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderNames,
				authorizationGET,
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../appliance/fixtures/stats_appliance.json"),
				},
			},
			askStubs: func(s *prompt.AskStubber) {
				s.StubPrompt("Username:").AnswerWith("bob")
				s.StubPrompt("Password:").AnswerWith("alice")
			},
		},
		{
			name: "signin prompt username and password and MFA token",
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderNames,
				{
					URL: "/authorization",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							if v, ok := r.Header["Authorization"]; ok && v[0] == "Bearer newToken" {
								rw.WriteHeader(http.StatusOK)
								fmt.Fprint(rw, string(`{
		                            "user": {
		                                "name": "admin",
		                                "needTwoFactorAuth": false,
		                                "canAccessAuditLogs": true,
		                                "privileges": [
		                                    {
		                                        "type": "All",
		                                        "target": "All",
		                                        "scope": {
		                                            "all": true,
		                                            "ids": [],
		                                            "tags": []
		                                        }
		                                    }
		                                ]
		                            },
		                            "token": "authorizedNewToken",
		                            "expires": "2022-02-01T15:07:04.451882Z"
		                        }`))
								return
							}
							rw.WriteHeader(http.StatusPreconditionFailed)
							fmt.Fprint(rw, string(`{
		                        "id": "precondition failed",
		                        "message": "Administrative authorization requires two-factor authentication.",
		                        "otpRequired": true,
		                        "username": "admin"
		                    }`))
							return
						}
					},
				},
				{
					URL: "/authentication/otp/initialize",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
		                        "inputType": "Numeric",
		                        "type": "AlreadySeeded"
		                    }`))
						}
					},
				},
				{
					URL: "/authentication/otp",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
		                        "user": {
		                            "name": "admin",
		                            "needTwoFactorAuth": false,
		                            "canAccessAuditLogs": false,
		                            "privileges": []
		                        },
		                        "token": "newToken",
		                        "expires": "2022-02-01T15:07:04.451882Z"
		                    }`))
						}
					},
				},
				{
					URL:       "/appliances",
					Responder: httpmock.JSONResponse("../appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/stats/appliances",
					Responder: httpmock.JSONResponse("../appliance/fixtures/stats_appliance.json"),
				},
			},
			askStubs: func(s *prompt.AskStubber) {
				s.StubPrompt("Username:").AnswerWith("bob")
				s.StubPrompt("Password:").AnswerWith("alice")
				s.StubPrompt("Please enter your one-time password:").AnswerWith("123456")
			},
		},
		{
			name:    "no auth no-interactive",
			wantErr: true,
			httpStubs: []httpmock.Stub{
				unauthorizedResponse,
				identityProviderNames,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyring.MockInit()
			registry := httpmock.NewRegistry(t)
			for _, v := range tt.httpStubs {
				registry.Register(v.URL, v.Responder)
			}
			for k, v := range tt.environmentVariables {
				t.Setenv(k, v)
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

			f := &factory.Factory{
				Config: &configuration.Config{
					Debug:                    false,
					URL:                      fmt.Sprintf("http://appgate.com:%d", registry.Port),
					PrimaryControllerVersion: "5.3.4-24950",
				},
				IOOutWriter: tty,
				Stdin:       pty,
				StdErr:      pty,
			}
			t.Cleanup(func() {
				if err := f.Config.ClearCredentials(); err != nil {
					t.Errorf("Failed to clear mock credentials after test %s", err)
				}
			})
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
				return a, nil
			}

			stubber, teardown := prompt.InitAskStubber(t)
			defer teardown()
			if tt.askStubs != nil {
				tt.askStubs(stubber)
				tt.askStubs = nil
			}
			if err := Signin(f); (err != nil) != tt.wantErr {
				t.Errorf("Signin() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := c.Tty().Close(); err != nil {
				t.Errorf("error closing Tty: %v", err)
			}
		})
	}
}
