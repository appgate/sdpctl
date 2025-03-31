package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/keyring"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	zkeyring "github.com/zalando/go-keyring"
)

var (
	authenticationResponse = httpmock.Stub{
		URL: "/admin/authentication",
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
		URL: "/admin/authentication",
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
		URL: "/admin/identity-providers/names",
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
	identityProviderMultipleNames = httpmock.Stub{
		URL: "/admin/identity-providers/names",
		Responder: func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "data": [
                        {
                            "authUrl": "https://idp.url/oauth2/v2.0/authorize",
                            "certificatePriorities": [],
                            "clientId": "xxxx",
                            "displayName": "AD OIDC",
                            "name": "AD OIDC",
                            "scope": "",
                            "tokenUrl": "https://idp.url/oauth2/v2.0/token",
                            "type": "Oidc"
                        },
                        {
                            "certificatePriorities": [],
                            "displayName": "AD SAML Admin",
                            "name": "SAML Admin",
                            "redirectUrl": "http://redirect.url",
                            "type": "Saml"
                        },
                        {
                            "certificatePriorities": [],
                            "displayName": "local",
                            "name": "local",
                            "type": "Credentials"
                        }
                    ]
                }
                `))
			}
		}}

	authorizationGET = httpmock.Stub{
		URL: "/admin/authorization",
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
                    "token": "VeryLongBearerTokenString",
                    "expires": "2019-08-24T14:15:22Z",
                    "messageOfTheDay": "Welcome to Appgate SDP."
                  }`))
			}
		}}
	authorizationGETNeedOTP = httpmock.Stub{
		URL: "/admin/authorization",
		Responder: func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusPreconditionFailed)
				fmt.Fprint(rw, string(`{
					"id": "precondition failed",
					"message": "Administrative authorization requires two-factor authentication.",
					"otpRequired": true,
					"username": "bob@appgate.com"
				}`))
			}
		}}

	authorizationInitAlreadySeeded = httpmock.Stub{
		URL: "/admin/authentication/otp/initialize",
		Responder: func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
					"inputType": "Numeric",
					"type": "AlreadySeeded"
				}`))
			}
		}}
	authorizationOtpAccepted = httpmock.Stub{
		URL: "/admin/authentication/otp",
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
		}}
	authorizationOtpDenied = httpmock.Stub{
		URL: "/admin/authentication/otp",
		Responder: func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(rw, string(`{
					"failureType": "Mfa",
					"id": "unauthorized",
					"message": "Invalid one-time password. Please make sure the time on your device is correct."
				}`))
			}
		}}
)

func TestSignProviderSelection(t *testing.T) {
	tests := []struct {
		name                 string
		askStubs             func(*prompt.PromptStubber)
		environmentVariables map[string]string
		httpStubs            []httpmock.Stub
		wantErr              bool
		disablePrompt        bool
		provider             *string
	}{
		{
			name: "Test with invalid provider name",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME": "bob",
				"SDPCTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderMultipleNames,
			},
			wantErr:  true,
			provider: openapi.PtrString("NotAValidProviderValue"),
		},
		{
			name: "Test with valid provider name",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME": "bob",
				"SDPCTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderMultipleNames,
				authorizationGET,
			},
			wantErr:  false,
			provider: openapi.PtrString("local"),
		},
		{
			name: "test with no provider set",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME": "bob",
				"SDPCTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderMultipleNames,
				authorizationGET,
			},
			wantErr:  false,
			provider: nil,
			askStubs: func(as *prompt.PromptStubber) {
				as.StubPrompt("Choose a provider:").AnswerWith("local")
			},
		},
		{
			name: "test with no provider set no prompt",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME": "bob",
				"SDPCTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderMultipleNames,
			},
			wantErr:       true,
			provider:      nil,
			disablePrompt: true,
		},
		{
			name: "test with provider and no prompt",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME": "bob",
				"SDPCTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderMultipleNames,
				authorizationGET,
			},
			wantErr:       false,
			provider:      openapi.PtrString("local"),
			disablePrompt: true,
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

			f := &factory.Factory{
				Config: &configuration.Config{
					Debug:    false,
					URL:      "http://localhost",
					Provider: tt.provider,
				},
				IOOutWriter: tty,
				Stdin:       pty,
				StdErr:      pty,
			}
			if tt.disablePrompt {
				f.DisablePrompt(true)
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registry.Client, nil
			}

			for k, v := range tt.environmentVariables {
				t.Setenv(k, v)
			}
			dir := t.TempDir()
			t.Setenv("SDPCTL_CONFIG_DIR", dir)

			stubber, teardown := prompt.InitStubbers(t)
			defer teardown()
			if tt.askStubs != nil {
				tt.askStubs(stubber)
				tt.askStubs = nil
			}
			if err := Signin(f); (err != nil) != tt.wantErr {
				if errors.Is(err, prompt.ErrNoPrompt) && tt.disablePrompt {
					t.Fatalf("got prompt for test %q has disabledPrompt %s", tt.name, err)
				}
				t.Fatal(err)
			}
			configFile := filepath.Join(dir, "config.json")
			if ok, err := util.FileExists(configFile); ok && err == nil {
				t.Fatalf("found config file %s but did not expect it", configFile)
			}
		})
	}
}
func TestSignInNoPromptOrEnv(t *testing.T) {
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   "http://localhost",
		},
		IOOutWriter: os.Stdout,
		Stdin:       os.Stdin,
		StdErr:      os.Stderr,
	}
	f.DisablePrompt(true)
	registry := httpmock.NewRegistry(t)
	registry.RegisterStub(identityProviderMultipleNames)
	registry.RegisterStub(unauthorizedResponse)
	defer registry.Teardown()
	registry.Serve()
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	err := Signin(f)
	if err == nil {
		t.Fatal("Expected err ErrSignInNotSupported")
	}
	if !errors.Is(ErrSignInNotSupported, err) {
		t.Errorf("expected %s, got %s", ErrSignInNotSupported, err)
	}
}

func TestSigninNoKeyringNoconfig(t *testing.T) {
	zkeyring.MockInit()
	registry := httpmock.NewRegistry(t)
	httpStubs := []httpmock.Stub{
		authenticationResponse,
		identityProviderNames,
		authorizationGET,
	}
	for _, v := range httpStubs {
		registry.Register(v.URL, v.Responder)
	}
	defer registry.Teardown()
	registry.Serve()
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   "http://localhost",
		},
		IOOutWriter: os.Stdout,
		Stdin:       os.Stdin,
		StdErr:      os.Stderr,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	environmentVariables := map[string]string{
		"SDPCTL_USERNAME":   "bob",
		"SDPCTL_PASSWORD":   "alice",
		"SDPCTL_NO_KEYRING": "true",
	}
	for k, v := range environmentVariables {
		t.Setenv(k, v)
	}
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)

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

	if err := Signin(f); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(dir, "config.json")
	if ok, err := util.FileExists(configFile); ok && err == nil {
		t.Fatalf("found config file %s but did not expect it", configFile)
	}

	prefix, err := f.Config.KeyringPrefix()
	if err != nil {
		t.Fatal("could not check verify keyring")
	}
	// make sure its not in keyring,
	// we will remove the env variable, if any to make sure we trigger
	// the keyring
	os.Unsetenv("SDPCTL_PASSWORD")
	if _, err := keyring.GetPassword(prefix); err != nil {
		t.Errorf("expected nil, got err %s", err)
	}
	os.Unsetenv("SDPCTL_USERNAME")
	if _, err := keyring.GetUsername(prefix); err != nil {
		t.Errorf("expected nil, got err %s", err)
	}
	os.Unsetenv("SDPCTL_BEARER")
	if _, err := keyring.GetBearer(prefix); err != nil {
		t.Errorf("expected nil, got err %s", err)
	}

	if _, err := f.Config.GetBearTokenHeaderValue(); err != nil {
		t.Fatalf("did not expect bearer token to be nil %s", err)
	}
}

func TestSignin(t *testing.T) {
	type testConfig struct {
		ExpiresAt   *string
		BearerToken *string
		Version     int
	}
	tests := []struct {
		name                 string
		askStubs             func(*prompt.PromptStubber)
		environmentVariables map[string]string
		testConfig           testConfig
		httpStubs            []httpmock.Stub
		wantErr              bool
	}{
		{
			name: "signin stored bearer token is valid",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME": "bob",
				"SDPCTL_PASSWORD": "alice",
			},
			httpStubs: []httpmock.Stub{
				authorizationGET,
			},
			testConfig: testConfig{
				ExpiresAt:   openapi.PtrString(time.Now().AddDate(0, 0, 3).String()),
				BearerToken: openapi.PtrString("MyExistingBearerToken"),
				Version:     18,
			},
		},
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
			},
		},
		{
			name: "signin prompt username and password",

			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderNames,
				authorizationGET,
			},
			askStubs: func(s *prompt.PromptStubber) {
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
					URL: "/admin/authorization",
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
				authorizationInitAlreadySeeded,
				authorizationOtpAccepted,
			},
			askStubs: func(s *prompt.PromptStubber) {
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
		{
			name: "no keyring",
			environmentVariables: map[string]string{
				"SDPCTL_USERNAME":   "bob",
				"SDPCTL_PASSWORD":   "alice",
				"SDPCTL_NO_KEYRING": "true",
			},
			wantErr: false,
			httpStubs: []httpmock.Stub{
				authenticationResponse,
				identityProviderNames,
				authorizationGET,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zkeyring.MockInit()
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
					Debug:       false,
					URL:         fmt.Sprintf("http://appgate.test:%d", registry.Port),
					ExpiresAt:   tt.testConfig.ExpiresAt,
					BearerToken: tt.testConfig.BearerToken,
					Version:     tt.testConfig.Version,
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

			stubber, teardown := prompt.InitStubbers(t)
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

func TestAuthAndOTP(t *testing.T) {
	type args struct {
		ctx      context.Context
		password *string
		token    string
	}
	tests := []struct {
		name       string
		args       args
		want       *string
		wantErr    bool
		httpStubs  []httpmock.Stub
		askStubs   func(*prompt.PromptStubber)
		wantErrOut *regexp.Regexp
	}{
		{
			name: "auth interactive OTP",
			args: args{
				ctx: context.Background(),
			},
			httpStubs: []httpmock.Stub{
				authorizationGETNeedOTP,
				authorizationInitAlreadySeeded,
				authorizationOtpAccepted,
			},
			askStubs: func(as *prompt.PromptStubber) {
				as.StubPrompt("Please enter your one-time password:").AnswerWith("12345")
			},
			wantErr: false,
			want:    openapi.PtrString("Bearer newToken"),
		},
		{
			name: "auth interactive OTP wrong OTPs",
			args: args{
				ctx: context.Background(),
			},
			httpStubs: []httpmock.Stub{
				authorizationGETNeedOTP,
				authorizationInitAlreadySeeded,
				authorizationOtpDenied,
			},
			askStubs: func(as *prompt.PromptStubber) {
				as.StubPrompt("Please enter your one-time password:").AnswerWith("99999")
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`OTP required`),
			want:       nil,
		},
		{
			name: "auth no interactive supported",
			args: args{
				ctx: context.WithValue(context.Background(), contextKeyCanPrompt, false),
			},
			httpStubs: []httpmock.Stub{
				authorizationGETNeedOTP,
			},
			wantErr:    true,
			want:       nil,
			wantErrOut: regexp.MustCompile(`authentication requires one-time-password, but a TTY prompt is not allowed, can't continue`),
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
			authenticator := NewAuth(registry.Client)
			stubber, teardown := prompt.InitStubbers(t)
			defer teardown()
			if tt.askStubs != nil {
				tt.askStubs(stubber)
				tt.askStubs = nil
			}

			got, err := authAndOTP(tt.args.ctx, authenticator, tt.args.password, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("authAndOTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Errorf("want %s got %s", tt.wantErrOut, err.Error())
				}
			}
			if tt.want != nil && *got != *tt.want {
				t.Errorf("authAndOTP() = %v, want %v", *got, *tt.want)
			}
		})
	}
}
