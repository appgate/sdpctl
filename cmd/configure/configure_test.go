package configure

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/spf13/viper"
)

func TestConfigCmd(t *testing.T) {
	defer viper.Reset()
	dir, err := os.MkdirTemp("", "sdpctl_test")
	if err != nil {
		t.Fatalf("cant create temp dir %s", err)
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

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	stubber, teardown := prompt.InitAskStubber(t)
	defer teardown()
	func(s *prompt.AskStubber) {
		s.StubPrompt("Enter the url for the controller API (example https://appgate.controller.com/admin)").
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
		t.Fatalf("cant create temp dir %s", err)
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
	cmd.SetArgs([]string{"--pem", "testdata/cert.pem"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	stubber, teardown := prompt.InitAskStubber(t)
	defer teardown()
	func(s *prompt.AskStubber) {
		s.StubPrompt("Enter the url for the controller API (example https://appgate.controller.com/admin)").
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
		t.Fatalf("cant create temp dir %s", err)
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

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	stubber, teardown := prompt.InitAskStubber(t)
	defer teardown()
	func(s *prompt.AskStubber) {
		s.StubPrompt("Enter the url for the controller API (example https://appgate.controller.com/admin)").
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
