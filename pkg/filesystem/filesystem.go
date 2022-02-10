package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	AgConfigDir   = "SDPCTL_CONFIG_DIR"
	XdgConfigHome = "XDG_CONFIG_HOME"
	AppData       = "AppData"
)

// ConfigDir path precedence
// 1. SDPCTL_CONFIG_DIR
// 2. XDG_CONFIG_HOME
// 3. AppData (windows only)
// 4. HOME
func ConfigDir() string {
	var path string
	name := "sdpctl"
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
