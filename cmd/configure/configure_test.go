package configure

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	pseudotty "github.com/creack/pty"
	"github.com/google/go-cmp/cmp"
	"github.com/hinshun/vt10x"
	"github.com/spf13/viper"
)

func TestConfigCmd(t *testing.T) {
	defer viper.Reset()
	dir, err := os.MkdirTemp("", "sdpctl_test")
	if err != nil {
		t.Fatalf("can't create temp dir %s", err)
	}
	defer os.RemoveAll(dir)
	viper.AddConfigPath(dir)
	viper.SetConfigType("json")
	if err := viper.SafeWriteConfig(); err != nil {
		t.Fatalf("test setup, write config failed %s", err)
	}

	pty, tty, err := pseudotty.Open()
	if err != nil {
		t.Fatalf("failed to open pseudotty: %s", err)
	}
	term := vt10x.New(vt10x.WithWriter(tty))
	c, err := expect.NewConsole(expect.WithStdin(pty), expect.WithStdout(term), expect.WithCloser(pty, tty))
	if err != nil {
		t.Fatalf("failed to create console: %s", err)
	}

	defer c.Close()

	f := &factory.Factory{
		Config:      &configuration.Config{},
		IOOutWriter: tty,
		Stdin:       pty,
		StdErr:      pty,
	}
	cmd := NewCmdConfigure(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "suppress interactive prompt with auto accept")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	stubber, teardown := prompt.InitAskStubber(t)
	defer teardown()
	func(s *prompt.AskStubber) {
		s.StubPrompt("Enter the url for the Controller API (example https://controller.company.com:8443)").
			AnswerWith("controller.appgate.com")

	}(stubber)

	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("execute configure command err %s", err)
	}

	configFile, err := os.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("could not open JSON config file %s", err)
	}
	defer configFile.Close()
	byteValue, err := io.ReadAll(configFile)
	if err != nil {
		t.Fatalf("could not read JSON config file %s", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(byteValue), &result); err != nil {
		t.Fatalf("could not json unmarshal config file %s", err)
	}
	v, ok := result["url"].(string)
	if !ok {
		t.Fatal("could not read url key from config")
	}
	want := "https://controller.appgate.com:8443/admin"
	if v != want {
		t.Fatalf("wrong addr stored in config, expected %q got %q", want, v)
	}
}

func TestConfigCmdWithPemFile(t *testing.T) {
	defer viper.Reset()
	dir, err := os.MkdirTemp("", "sdpctl_test")
	if err != nil {
		t.Fatalf("can't create temp dir %s", err)
	}
	defer os.RemoveAll(dir)
	viper.AddConfigPath(dir)
	viper.SetConfigType("json")
	if err := viper.SafeWriteConfig(); err != nil {
		t.Fatalf("test setup, write config failed %s", err)
	}

	pty, tty, err := pseudotty.Open()
	if err != nil {
		t.Fatalf("failed to open pseudotty: %s", err)
	}
	term := vt10x.New(vt10x.WithWriter(tty))
	c, err := expect.NewConsole(expect.WithStdin(pty), expect.WithStdout(term), expect.WithCloser(pty, tty))
	if err != nil {
		t.Fatalf("failed to create console: %s", err)
	}

	defer c.Close()

	f := &factory.Factory{
		Config:      &configuration.Config{},
		IOOutWriter: tty,
		Stdin:       pty,
		StdErr:      pty,
	}
	cmd := NewCmdConfigure(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "suppress interactive prompt with auto accept")
	cmd.SetArgs([]string{"--pem", "testdata/cert.pem"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	stubber, teardown := prompt.InitAskStubber(t)
	defer teardown()
	func(s *prompt.AskStubber) {
		s.StubPrompt("Enter the url for the Controller API (example https://controller.company.com:8443)").
			AnswerWith("another.appgate.com")

	}(stubber)

	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("execute configure command err %s", err)
	}

	configFile, err := os.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("could not open JSON config file %s", err)
	}
	defer configFile.Close()
	byteValue, err := io.ReadAll(configFile)
	if err != nil {
		t.Fatalf("could not read JSON config file %s", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(byteValue), &result); err != nil {
		t.Fatalf("could not json unmarshal config file %s", err)
	}
	v, ok := result["url"].(string)
	if !ok {
		t.Fatal("could not read url key from config")
	}
	want := "https://another.appgate.com:8443/admin"
	if v != want {
		t.Fatalf("wrong addr stored in config, expected %q got %q", want, v)
	}
	t.Logf("Computed config %+v\n", result)
	pem, ok := result["pem_filepath"].(string)
	if !ok {
		t.Fatal("could not read pem_filepath key from config")
	}
	if !strings.HasSuffix(pem, "testdata/cert.pem") {
		t.Fatalf("pem suffix value wrong, got %s, expected %s", pem, "testdata/cert.pem")
	}
}

func TestConfigCmdWithExistingAddr(t *testing.T) {
	defer viper.Reset()
	dir, err := os.MkdirTemp("", "sdpctl_test*")
	if err != nil {
		t.Fatalf("can't create temp dir %s", err)
	}
	defer os.RemoveAll(dir)
	configPath := filepath.Join(dir, "config.json")

	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("test setup failed, %s", err)
	}
	file.Close()

	data := []byte(`{
        "url": "https://foobar.com"
    }`)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("test setup, write existing config failed %s", err)
	}

	preConfigFile, err := os.Open(configPath)
	if err != nil {
		t.Fatalf("could not open JSON config file %s", err)
	}
	defer preConfigFile.Close()

	viper.AddConfigPath(dir)
	viper.SetConfigType("json")

	pty, tty, err := pseudotty.Open()
	if err != nil {
		t.Fatalf("failed to open pseudotty: %s", err)
	}
	term := vt10x.New(vt10x.WithWriter(tty))
	c, err := expect.NewConsole(expect.WithStdin(pty), expect.WithStdout(term), expect.WithCloser(pty, tty))
	if err != nil {
		t.Fatalf("failed to create console: %s", err)
	}

	defer c.Close()

	f := &factory.Factory{
		Config:      &configuration.Config{},
		IOOutWriter: tty,
		Stdin:       pty,
		StdErr:      pty,
	}
	cmd := NewCmdConfigure(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "suppress interactive prompt with auto accept")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	stubber, teardown := prompt.InitAskStubber(t)
	defer teardown()
	func(s *prompt.AskStubber) {
		s.StubPrompt("Enter the url for the Controller API (example https://controller.company.com:8443)").
			AnswerWith("new.appgate.com")

	}(stubber)

	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("execute configure command err %s", err)
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		t.Fatalf("could not open JSON config file %s", err)
	}
	defer configFile.Close()
	byteValue, err := io.ReadAll(configFile)
	if err != nil {
		t.Fatalf("could not read JSON config file %s", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(byteValue), &result); err != nil {
		t.Fatalf("could not json unmarshal config file %s", err)
	}
	v, ok := result["url"].(string)
	if !ok {
		t.Fatal("could not read url key from config")
	}
	want := "https://new.appgate.com:8443/admin"
	if v != want {
		t.Fatalf("wrong addr stored in config, expected %q got %q", want, v)
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
			got := certificateDetails(tt.cert)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
