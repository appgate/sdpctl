package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	BearerToken string `mapstructure:"bearer"`
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
