package configuration

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	AgConfigDir   = "APPGATECTL_CONFIG_DIR"
	XdgConfigHome = "XDG_CONFIG_HOME"
	AppData       = "AppData"
)

type Config struct {
	URL             string `mapstructure:"url"`
	Provider        string
	Insecure        bool
	Debug           bool   // http debug flag
	Version         int    `mapstructure:"api_version"` // api peer interface version
	BearerToken     string `mapstructure:"bearer"`      // current logged in user token
	ExpiresAt       string `mapstructure:"expires_at"`
	CredentialsFile string `mapstructure:"credentials_file"`
	DeviceID        string `mapstructure:"device_id"`
	PemFilePath     string `mapstructure:"pem_filepath"`
}

type Credentials struct {
	Username string
	Password string
}

func (c *Config) GetBearTokenHeaderValue() string {
	return fmt.Sprintf("Bearer %s", c.BearerToken)
}

// ConfigDir path precedence
// 1. APPGATECTL_CONFIG_DIR
// 2. XDG_CONFIG_HOME
// 3. AppData (windows only)
// 4. HOME
func ConfigDir() string {
	var path string
	name := "appgatectl"
	if a := os.Getenv(AgConfigDir); a != "" {
		path = a
	} else if b := os.Getenv(XdgConfigHome); b != "" {
		path = filepath.Join(b, name)
	} else if c := os.Getenv(AppData); runtime.GOOS == "windows" && c != "" {
		path = filepath.Join(c, name)
	} else {
		d, _ := os.UserHomeDir()
		path = filepath.Join(d, ".config", name)
	}

	return path
}

func IsAuthCheckEnabled(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return false
	}
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations["skipAuthCheck"] == "true" {
			return false
		}
	}
	return true
}

func (c *Config) CheckAuth() bool {
	layout := "2006-01-02 15:04:05.999999999 -0700 MST"
	d, err := time.Parse(layout, c.ExpiresAt)
	if err != nil {
		return false
	}
	if len(c.BearerToken) < 1 {
		return false
	}
	if len(c.URL) < 1 {
		return false
	}
	if len(c.Provider) < 1 {
		return false
	}
	t1 := time.Now()
	return t1.Before(d)
}

func (c *Config) LoadCredentials() (*Credentials, error) {
	creds := &Credentials{}

	// No file is set so we return empty credentials
	if len(c.CredentialsFile) <= 0 {
		return creds, nil
	}

	// File is set in the config, but does not exists, so we return empty credentials
	info, err := os.Stat(c.CredentialsFile)
	if err != nil && os.IsNotExist(err) {
		return creds, nil
	}

	// Check file permissions
	// If file exists, it should only be readable by the executing user
	mode := info.Mode()
	if mode&(1<<2) != 0 {
		return nil, errors.New("invalid permissions on credentials file")
	}

	// Scan file for credentials
	file, err := os.Open(c.CredentialsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := strings.Split(scanner.Text(), "=")
		switch data[0] {
		case "username":
			creds.Username = data[1]
		case "password":
			creds.Password = data[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return creds, nil
}

func (c *Config) StoreCredentials(crd *Credentials) error {
	joinStrings := []string{}
	if len(crd.Username) > 0 {
		joinStrings = append(joinStrings, fmt.Sprintf("username=%s", crd.Username))
	}
	if len(crd.Password) > 0 {
		joinStrings = append(joinStrings, fmt.Sprintf("password=%s", crd.Password))
	}
	b := []byte(strings.Join(joinStrings, "\n"))

	path := filepath.FromSlash(c.CredentialsFile)
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, b, 0600)
	if err != nil {
		return err
	}
	log.WithField("path", c.CredentialsFile).Info("Stored credentials")
	viper.Set("credentials_file", c.CredentialsFile)
	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) GetHost() (string, error) {
	url, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}
	return url.Hostname(), nil
}

func (c *Config) GetPort() (string, error) {
	url, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}
	return url.Port(), nil
}

func (c *Config) GetScheme() (string, error) {
	url, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}
	return url.Scheme, nil
}
