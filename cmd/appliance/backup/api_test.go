package backup

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

func TestBackupAPICommandAlreadyEnabled(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/global-settings",
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
	cmd.PersistentFlags().Bool("no-interactive", false, "")
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
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/global-settings",
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
	cmd := NewBackupAPICmd(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	stubber, teardown := prompt.InitAskStubber(t)
	defer teardown()
	func(prompt *prompt.AskStubber) {
		prompt.StubPrompt("The passphrase to encrypt the appliance backups when the Backup API is used:").AnswerWith("secret")
		prompt.StubPrompt("Confirm your passphrase:").AnswerWith("secret")
	}(stubber)

	_, err = cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	want := regexp.MustCompile(`The Backup API and the passphrase have been updated`)
	if !want.MatchString(string(got)) {
		t.Fatalf("Expected output\n%s\ngot\n%s\n", want, got)
	}
}
