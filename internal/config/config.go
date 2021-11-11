package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	AgConfigDir   = "APPGATECTL_CONFIG_DIR"
	XdgConfigHome = "XDG_CONFIG_HOME"
	AppData       = "AppData"
)

type Config struct {
	URL         string `mapstructure:"url"`
	Provider    string
	Insecure    bool
	Debug       bool   // http debug flag
	Version     int    `mapstructure:"api_version"` // api peer interface version
	BearerToken string `mapstructure:"bearer"`      // current logged in user token
	ExpiresAt   string `mapstructure:"expires_at"`
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

func (c *Config) Validate() error {
	if c.BearerToken == "" || c.ExpiresAt == "" {
		return fmt.Errorf("Invalid user session. Please use 'appgatectl configure login'")
	}

	layout := "2006-01-02 15:04:05.000000 +0000 UTC"
	expDate, err := time.Parse(layout, c.ExpiresAt)
	if err != nil {
		return err
	}

	// Check expiration date
	if time.Now().After(expDate) {
		return fmt.Errorf("Session expired. Please use 'appgatectl configure login' to log in again")
	}

	// TODO: Validate token

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
