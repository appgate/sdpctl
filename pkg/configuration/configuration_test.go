package configuration

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/appgate/sdpctl/pkg/keyring"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	zkeyring "github.com/zalando/go-keyring"
)

func TestConfigCheckAuth(t *testing.T) {
	zkeyring.MockInit()
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)
	if err := keyring.SetBearer("controller.appgate.com", "abc123456789"); err != nil {
		t.Fatalf("unable to mock keyring in TestConfigCheckAuth() %v", err)
	}
	type fields struct {
		URL             string
		Provider        string
		Insecure        bool
		Debug           bool
		Version         int
		BearerToken     string
		ExpiresAt       string
		CredentialsFile string
		DeviceID        string
		PemFilePath     string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "valid",
			fields: fields{
				ExpiresAt: "2031-12-08 08:15:39.137584 +0000 UTC",
				URL:       "https://controller.appgate.com",
				Provider:  "local",
			},
			want: false,
		},
		{
			name: "invalid expire date",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
				URL:       "https://controller.appgate.com",
				Provider:  "local",
			},
			want: true,
		},
		{
			name: "no token",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
			},
			want: true,
		},
		{
			name: "no url",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
				Provider:  "local",
			},
			want: true,
		},
		{
			name: "no provider",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
				URL:       "https://controller.appgate.com",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				URL:         tt.fields.URL,
				Provider:    &tt.fields.Provider,
				Insecure:    tt.fields.Insecure,
				Debug:       tt.fields.Debug,
				Version:     tt.fields.Version,
				BearerToken: &tt.fields.BearerToken,
				ExpiresAt:   &tt.fields.ExpiresAt,
				DeviceID:    tt.fields.DeviceID,
				PemFilePath: tt.fields.PemFilePath,
			}
			if got := c.IsRequireAuthentication(); got != tt.want {
				t.Errorf("Config.IsRequireAuthentication() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigGetHost(t *testing.T) {
	type fields struct {
		URL string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "valid URL",
			fields: fields{
				URL: "http://controller.com/admin",
			},
			want:    "controller.com",
			wantErr: false,
		},
		{
			name: "ipv6 addr",
			fields: fields{
				URL: "http://[fd00:ffff:a:93:172:17:93:35]:666/admin",
			},
			want:    "fd00:ffff:a:93:172:17:93:35",
			wantErr: false,
		},
		{
			name: "empty URL",
			fields: fields{
				URL: "",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				URL: tt.fields.URL,
			}
			got, err := c.GetHost()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.GetHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.GetHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		Name string
		URL  string
		want string
		err  bool
	}{
		{
			Name: "Full valid URL",
			URL:  "https://some.valid.url:8443/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No scheme",
			URL:  "some.valid.url:8443/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "HTTP scheme",
			URL:  "http://some.valid.url:8443/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No path",
			URL:  "https://some.valid.url:8443",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No port",
			URL:  "https://some.valid.url/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No port and path",
			URL:  "https://some.valid.url",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No port, path or scheme",
			URL:  "some.valid.url",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No scheme or port",
			URL:  "some.valid.url/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "Other port",
			URL:  "https://some.valid.url:443/admin",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "Other port, no path",
			URL:  "https://some.valid.url:443",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "Other port, no path, no scheme",
			URL:  "some.valid.url:443",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "Other port, no path, no scheme",
			URL:  "some.valid.url:443",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "No URL",
			URL:  "",
			want: "",
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result, err := NormalizeConfigurationURL(tt.URL)
			if err != nil && !tt.err {
				t.Fatalf("Test failed. Error: %v", err)
			}

			if result != tt.want {
				t.Fatalf("FAILED! EXPECTED: %s, GOT: %s", tt.want, result)
			}
		})
	}
}

func TestClearCredentials(t *testing.T) {
	zkeyring.MockInit()
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)
	var (
		prefix   = "test-unit.devops"
		username = "user"
		password = "password"
		bearer   = "somebearer"
	)
	cfg := Config{
		URL:         fmt.Sprintf("https://%s", prefix),
		BearerToken: &bearer,
	}
	if err := keyring.SetUsername(prefix, username); err != nil {
		t.Error("TEST FAIL: failed to set username", err)
	}
	if err := keyring.SetPassword(prefix, password); err != nil {
		t.Error("TEST FAIL: failed to set password", err)
	}
	if err := keyring.SetBearer(prefix, bearer); err != nil {
		t.Error("TEST FAIL: failed to set bearer", err)
	}

	if err := cfg.ClearCredentials(); err != nil {
		t.Error("TEST FAIL: ClearCredentials() failed:", err)
	}

	if _, err := keyring.GetUsername(prefix); err == nil {
		t.Error("TEST FAIL: failed to remove username")
	}
	if _, err := keyring.GetPassword(prefix); err == nil {
		t.Error("TEST FAIL: failed to remove password")
	}
	if _, err := keyring.GetBearer(prefix); err == nil {
		t.Error("TEST FAIL: failed to remove bearer")
	}
}

func TestCheckAPIVersionRestriction(t *testing.T) {
	type args struct {
		cmd        *cobra.Command
		apiVersion int
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantErrOut *regexp.Regexp
	}{
		{
			name: "version is ok",
			args: args{
				cmd: &cobra.Command{
					Use: "testCommand",
					Annotations: map[string]string{
						"MinAPIVersion": "17",
					},
				},
				apiVersion: 17,
			},
			wantErr: false,
		},
		{
			name: "version too low",
			args: args{
				cmd: &cobra.Command{
					Use: "testCommand",
					Annotations: map[string]string{
						"MinAPIVersion": "17",
					},
				},
				apiVersion: 16,
			},
			wantErr: true,
		},
		{
			name: "custom error message",
			args: args{
				cmd: &cobra.Command{
					Use: "testCommand",
					Annotations: map[string]string{
						"MinAPIVersion": "17",
						"ErrorMessage":  "this command is not supported on v17",
					},
				},
				apiVersion: 16,
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`this command is not supported on v17`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckMinAPIVersionRestriction(tt.args.cmd, tt.args.apiVersion)
			if (err != nil) != tt.wantErr || tt.wantErrOut != nil {
				if tt.wantErrOut != nil {
					if !tt.wantErrOut.MatchString(err.Error()) {
						t.Fatalf("expected error message to match, want %s, got %s", tt.wantErrOut, err.Error())
					}
				} else {
					t.Errorf("CheckAPIVersionRestriction() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

var demoCert = `-----BEGIN CERTIFICATE-----
MIIB4TCCAYugAwIBAgIUblfrUTadV6hHYGW8B/T0kVve6GAwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMTEyMDYxMDI2NTVaFw0zMTEy
MDQxMDI2NTVaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwXDANBgkqhkiG9w0BAQEF
AANLADBIAkEAyu++YjSfKQW7DfYmKQbEIG3TyD91Cce1VBVg+KwLP/iBNLQO1ZFR
gYoiQRHqOH9iHOZRfJBhZiAB7MSxDuIdrwIDAQABo1MwUTAdBgNVHQ4EFgQU/iVT
noAPQ09G4sC26jHKu0xnsXQwHwYDVR0jBBgwFoAU/iVTnoAPQ09G4sC26jHKu0xn
sXQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAANBADwEHw0k7sUuIetl
YdaOvNqyH5SnPUDncp4Gkpr61rpVQzwadnCTtiAisYor+gD1lehtj/AjZMxvJdOm
K0mfdZQ=
-----END CERTIFICATE-----`

func Test_certificateDetails(t *testing.T) {
	before, _ := time.Parse("2006-01-02 15:04", "2018-01-20 04:35")
	after, _ := time.Parse("2006-01-02 15:04", "2024-01-20 04:35")

	tests := []struct {
		name string
		cert *x509.Certificate
		want string
	}{
		{
			name: "bla",
			cert: &x509.Certificate{
				Raw:       []byte(demoCert),
				NotBefore: before,
				NotAfter:  after,
				Subject: pkix.Name{
					CommonName: "controller.appgate.com",
				},
				Issuer: pkix.Name{
					CommonName: "Appgate SDP CA",
				},
			},
			want: `[Subject]
	controller.appgate.com
[Issuer]
	Appgate SDP CA
[Not Before]
	2018-01-20 04:35:00 +0000 UTC
[Not After]
	2024-01-20 04:35:00 +0000 UTC
[Thumbprint SHA-1]
	00:2E:E6:59:93:63:70:E9:50:7B:90:70:9F:4B:58:D3:30:E5:B5:F5
[Thumbprint SHA-256]
	8E:31:BA:3F:9E:06:9F:A1:86:5A:2E:14:58:84:C9:7E:23:51:93:8D:92:F3:A8:9E:EE:BC:FC:11:AD:DF:12:1C
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CertificateDetails(tt.cert)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
