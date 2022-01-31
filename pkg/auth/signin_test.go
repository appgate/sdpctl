package auth

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
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

	identityProviderNnames = httpmock.Stub{
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

func TestSignin(t *testing.T) {
	type args struct {
		remember   bool
		saveConfig bool
	}
	tests := []struct {
		name                 string
		args                 args
		askStubs             func(*prompt.AskStubber)
		environmentVariables map[string]string
		httpStubs            []httpmock.Stub
		wantErr              bool
	}{
		{
			name: "signin with environment variables",
			args: args{
				remember:   false,
				saveConfig: false,
			},
			environmentVariables: map[string]string{
				"APPGATECTL_USERNAME": "bob",
				"APPGATECTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderNnames,
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
			args: args{
				remember:   false,
				saveConfig: false,
			},

			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderNnames,
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
				s.StubOne("bob")   // username
				s.StubOne("alice") // password
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := httpmock.NewRegistry()
			for _, v := range tt.httpStubs {
				registry.Register(v.URL, v.Responder)
			}
			for k, v := range tt.environmentVariables {
				os.Setenv(k, v)
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
				return a, nil
			}

			stubber, teardown := prompt.InitAskStubber()
			defer teardown()

			if tt.askStubs != nil {
				tt.askStubs(stubber)
			}
			if err := Signin(f, tt.args.remember, tt.args.saveConfig); (err != nil) != tt.wantErr {
				t.Errorf("Signin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
