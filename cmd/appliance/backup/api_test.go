package backup

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
)

func TestBackupAPICommandAlreadyEnabled(t *testing.T) {
	registry := httpmock.NewRegistry()
	registry.Register(
		"/global-settings",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "claimsTokenExpiration": 1440,
                    "entitlementTokenExpiration": 180,
                    "administrationTokenExpiration": 720,
                    "vpnCertificateExpiration": 525600,
                    "spaMode": "TCP",
                    "loginBannerMessage": "Authorized use only.",
                    "messageOfTheDay": "Welcome to Appgate SDP.",
                    "backupApiEnabled": true,
                    "fips": false,
                    "geoIpUpdates": false,
                    "auditLogPersistenceMode": "Default",
                    "appDiscoveryDomains": [
                      "company.com"
                    ],
                    "collectiveId": "4c07bc69-57ea-42dd-b702-c2d6c45419fc"
                  }
                `))
			}
		},
	)
	defer registry.Teardown()
	registry.Serve()
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
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
	cmd := NewBackupAPICmd(f)

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	want := regexp.MustCompile(`Backup API is already enabled`)
	if !want.MatchString(string(got)) {
		t.Fatalf("Expected output\n%s\ngot\n%s\n", want, got)
	}
}

func TestBackupAPICommand(t *testing.T) {
	registry := httpmock.NewRegistry()
	registry.Register(
		"/global-settings",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "claimsTokenExpiration": 1440,
                    "entitlementTokenExpiration": 180,
                    "administrationTokenExpiration": 720,
                    "vpnCertificateExpiration": 525600,
                    "spaMode": "TCP",
                    "loginBannerMessage": "Authorized use only.",
                    "messageOfTheDay": "Welcome to Appgate SDP.",
                    "backupApiEnabled": false,
                    "fips": false,
                    "geoIpUpdates": false,
                    "auditLogPersistenceMode": "Default",
                    "appDiscoveryDomains": [
                      "company.com"
                    ],
                    "collectiveId": "4c07bc69-57ea-42dd-b702-c2d6c45419fc"
                  }
                `))
			}
		},
	)
	defer registry.Teardown()
	registry.Serve()
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
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
	cmd := NewBackupAPICmd(f)

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	stubber, teardown := prompt.InitAskStubber()
	defer teardown()
	func(prompt *prompt.AskStubber) {
		prompt.StubOne("newBackupPassphrase") // password
		prompt.StubOne("newBackupPassphrase") // password confirmation
	}(stubber)

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	want := regexp.MustCompile(`Backup API and phassphrase has been updated`)
	if !want.MatchString(string(got)) {
		t.Fatalf("Expected output\n%s\ngot\n%s\n", want, got)
	}
}
