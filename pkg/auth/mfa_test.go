package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/go-cmp/cmp"
)

func TestAuthProviderNames(t *testing.T) {
	tests := []struct {
		name      string
		httpStubs []httpmock.Stub
		want      []openapi.IdentityProvidersNamesGet200ResponseDataInner
		wantErr   bool
	}{
		{
			name: "list providers",
			httpStubs: []httpmock.Stub{
				{
					URL: "/identity-providers/names",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
                                "data": [
                                    {
                                        "name": "SAML Admin",
                                        "displayName": "SAML Admin",
                                        "type": "Saml",
                                        "redirectUrl": "https://login.microsoftonline.com/321312",
                                        "certificatePriorities": []
                                    },
                                    {
                                        "name": "local",
                                        "displayName": "local",
                                        "type": "Credentials",
                                        "certificatePriorities": []
                                    }
                                ],
                                "bannerMessage": "Authorized use only"
                            }`))
						}
					},
				},
			},
			want: []openapi.IdentityProvidersNamesGet200ResponseDataInner{
				{
					Name:                  openapi.PtrString("SAML Admin"),
					RedirectUrl:           openapi.PtrString("https://login.microsoftonline.com/321312"),
					Type:                  openapi.PtrString("Saml"),
					CertificatePriorities: []map[string]interface{}{},
				},
				{
					Name:                  openapi.PtrString("local"),
					Type:                  openapi.PtrString("Credentials"),
					CertificatePriorities: []map[string]interface{}{},
				},
			},
			wantErr: false,
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
			a := &Auth{
				APIClient: registry.Client,
			}
			got, err := a.ProviderNames(context.TODO())
			if (err != nil) != tt.wantErr {
				t.Errorf("Auth.ProviderNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("Auth.ProviderNames() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

const successfulLoginResponse = `{
    "version": "4.3.0-20000",
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
    "token": "423432423423432432423",
    "expires": "2019-08-24T14:15:22Z",
    "messageOfTheDay": "Welcome to Appgate SDP."
  }`

func TestAuthAuthentication(t *testing.T) {
	tests := []struct {
		name       string
		httpStubs  []httpmock.Stub
		signinOpts openapi.LoginRequest
		wantToken  string
		wantErr    bool
	}{
		{
			name: "authentication OK",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authentication",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(successfulLoginResponse))
						}
					},
				},
			},
			signinOpts: openapi.LoginRequest{
				ProviderName: "local",
				Username:     openapi.PtrString("user"),
				Password:     openapi.PtrString("tSW3!QBv(rj{UuLY"),
				DeviceId:     "4c07bc67-57ea-42dd-b702-c2d6c45419fc",
			},
			wantToken: "423432423423432432423",
			wantErr:   false,
		},
		{
			name: "authentication 406",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authentication",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusNotAcceptable)
							fmt.Fprint(rw, string(`{
                                "id": "not acceptable",
                                "message": "Invalid 'Accept' header. Current version: application/vnd.appgate.peer-v15+json, Received: application/vnd.appgate.peer-v1+json",
                                "minSupportedVersion": 7,
                                "maxSupportedVersion": 15
                            }`))
						}
					},
				},
			},
			signinOpts: openapi.LoginRequest{
				ProviderName: "local",
				Username:     openapi.PtrString("user"),
				Password:     openapi.PtrString("tSW3!QBv(rj{UuLY"),
				DeviceId:     "4c07bc67-57ea-42dd-b702-c2d6c45419fc",
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "authentication 500",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authentication",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusInternalServerError)
							fmt.Fprint(rw, string(`{
                                "id": "string",
                                "message": "string"
                              }`))
						}
					},
				},
			},
			signinOpts: openapi.LoginRequest{
				ProviderName: "local",
				Username:     openapi.PtrString("user"),
				Password:     openapi.PtrString("tSW3!QBv(rj{UuLY"),
				DeviceId:     "4c07bc67-57ea-42dd-b702-c2d6c45419fc",
			},
			wantToken: "",
			wantErr:   true,
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
			a := &Auth{
				APIClient: registry.Client,
			}
			got, minMax, err := a.Authentication(context.TODO(), tt.signinOpts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Auth.Authentication() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.GetToken() != tt.wantToken {
				t.Fatalf("got Token %s, expected %s", got.GetToken(), tt.wantToken)
			}
			if tt.wantErr && minMax != nil {
				if minMax.Max != 15 {
					t.Fatalf("Max peer version invalid got %+v", minMax)
				}
				if minMax.Min != 7 {
					t.Fatalf("Min peer version invalid got %+v", minMax)
				}

			}
		})
	}
}

func TestAuthAuthorization(t *testing.T) {
	tests := []struct {
		name       string
		httpStubs  []httpmock.Stub
		signinOpts openapi.LoginRequest
		wantToken  string
		wantErr    bool
	}{
		{
			name: "Authorization OK",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authorization",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(successfulLoginResponse))
						}
					},
				},
			},
			signinOpts: openapi.LoginRequest{
				ProviderName: "local",
				Username:     openapi.PtrString("user"),
				Password:     openapi.PtrString("tSW3!QBv(rj{UuLY"),
				DeviceId:     "4c07bc67-57ea-42dd-b702-c2d6c45419fc",
			},
			wantToken: "423432423423432432423",
			wantErr:   false,
		},
		{
			name: "Authorization 412",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authorization",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusPreconditionFailed)
							fmt.Fprint(rw, string(`{
                                "id": "precondition failed",
                                "message": "Administrative authorization requires two-factor authentication.",
                                "otpRequired": true,
                                "username": "admin"
                            }`))
						}
					},
				},
			},
			signinOpts: openapi.LoginRequest{
				ProviderName: "local",
				Username:     openapi.PtrString("user"),
				Password:     openapi.PtrString("tSW3!QBv(rj{UuLY"),
				DeviceId:     "4c07bc67-57ea-42dd-b702-c2d6c45419fc",
			},
			wantToken: "abc321",
			wantErr:   true,
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
			a := &Auth{
				APIClient: registry.Client,
			}
			got, err := a.Authorization(context.TODO(), "abc123")
			if (err != nil) != tt.wantErr {
				if !errors.Is(err, ErrPreConditionFailed) {
					t.Fatalf("Expected ErrPreConditionFailed, got %s", err)
				}
				t.Errorf("Auth.Authorization() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.GetToken() != tt.wantToken {
				t.Fatalf("got Token %s, expected %s", got.GetToken(), tt.wantToken)
			}
		})
	}
}

func TestAuthInitializeOTP(t *testing.T) {
	tests := []struct {
		name       string
		httpStubs  []httpmock.Stub
		typeMethod string
		wantErr    bool
	}{
		{
			name: "otp initialize already seeded",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authentication/otp/initialize",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{
                                "type": "AlreadySeeded",
                                "secret": "6XOEKS6WZASFPA5A",
                                "otpAuthUrl": "otpauth://totp/admin@local@appgate.company.com?secret=6XOEKS6WZASFPA5A&issuer=Appgate%20SDP",
                                "barcode": "string",
                                "responseMessage": "Please enter enter 1234 to your token.",
                                "state": "string",
                                "timeout": 10,
                                "sendPassword": true
                              }`))
						}
					},
				},
			},
			typeMethod: "AlreadySeeded",
			wantErr:    false,
		},
		{
			name: "otp initialize 422",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authentication/otp/initialize",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusUnprocessableEntity)
							fmt.Fprint(rw, string(`{
                                "id": "string",
                                "message": "string",
                                "errors": [
                                  {
                                    "field": "name",
                                    "message": "may not be null"
                                  }
                                ]
                              }`))
						}
					},
				},
			},
			wantErr: true,
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
			a := &Auth{
				APIClient: registry.Client,
			}
			got, err := a.InitializeOTP(context.TODO(), openapi.PtrString("password"), "token")
			if (err != nil) != tt.wantErr {
				t.Errorf("Auth.InitializeOTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && !strings.Contains(err.Error(), "name may not be null") {
				t.Fatalf("Invalid error message, got %s", err)
			}
			if got.GetType() != tt.typeMethod {
				t.Fatalf("Expected %s, got %s", tt.typeMethod, got.GetType())
			}
		})
	}
}

func TestAuthPushOTP(t *testing.T) {
	tests := []struct {
		name      string
		httpStubs []httpmock.Stub
		wantErr   bool
	}{
		{
			name: "otp initialize already seeded",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authentication/otp",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(successfulLoginResponse))
						}
					},
				},
			},
			wantErr: false,
		},
		{
			name: "otp initialize invalid token",
			httpStubs: []httpmock.Stub{
				{
					URL: "/authentication/otp",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusUnauthorized)
							fmt.Fprint(rw, string(`{
                                "id": "unauthorized",
                                "message": "Invalid one-time password. Please make sure the time on your device is correct.",
                                "failureType": "Mfa"
                            }`))
						}
					},
				},
			},
			wantErr: true,
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
			a := &Auth{
				APIClient: registry.Client,
			}
			got, err := a.PushOTP(context.TODO(), "987654", "token")
			if (err != nil) != tt.wantErr {
				t.Errorf("Auth.PushOTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && err.Error() != "Invalid one-time password" {
				t.Fatalf("Invalid error message, got %s", err)
			}
			t.Logf("Got %+v", got)
		})
	}
}
