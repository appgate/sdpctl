package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigDir(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		onlyWindows bool
		env         map[string]string
		output      string
	}{
		{
			name: "HOME/USERPROFILE specified",
			env: map[string]string{
				"SDPCTL_CONFIG_DIR": "",
				"XDG_CONFIG_HOME":   "",
				"AppData":           "",
				"HOME":              tempDir,
			},
			output: filepath.Join(tempDir, ".config", "sdpctl"),
		},
		{
			name: "SDPCTL_CONFIG_DIR specified",
			env: map[string]string{
				"SDPCTL_CONFIG_DIR": filepath.Join(tempDir, "sdpctl_dir"),
			},
			output: filepath.Join(tempDir, "sdpctl_dir"),
		},
		{
			name: "XDG_CONFIG_HOME specified",
			env: map[string]string{
				"XDG_CONFIG_HOME": tempDir,
			},
			output: filepath.Join(tempDir, "sdpctl"),
		},
	}

	for _, tt := range tests {
		if tt.onlyWindows && runtime.GOOS != "windows" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != nil {
				for k, v := range tt.env {
					old := os.Getenv(k)
					os.Setenv(k, v)
					defer os.Setenv(k, old)
				}
			}

			if dir := ConfigDir(); dir != tt.output {
				t.Errorf("Got %s, expected %s", tt.output, dir)
			}
		})
	}
}
