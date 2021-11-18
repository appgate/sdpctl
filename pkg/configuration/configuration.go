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

	"github.com/spf13/cobra"
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
}

type Credentials struct {
	username string
	password string
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

func (c *Config) GetCredentialsFromFile() (*Credentials, error) {
    if len(c.CredentialsFile) <= 0 {
        return nil, errors.New("no credentials file set")
    }

	creds := &Credentials{}

	// Check file permissions
	info, err := os.Stat(c.CredentialsFile)
	if err != nil {
		return nil, err
	}
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
			creds.username = data[1]
		case "password":
			creds.password = data[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Check if all required fields are set
	if len(creds.username) <= 0 || len(creds.password) <= 0 {
		return nil, errors.New("invalid credentials")
	}

	return creds, nil
}

func (c *Config) GetCredentialsFromEnv(uEnv string, pEnv string) (*Credentials, error) {
	creds := &Credentials{}

	creds.username = os.Getenv(uEnv)
	creds.password = os.Getenv(pEnv)

	if len(creds.username) <= 0 || len(creds.password) <= 0 {
		return nil, errors.New("invalid credentials")
	}

	return creds, nil
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
