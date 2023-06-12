package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/adrg/xdg"
)

func TestConfigDir(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		onlyWindows bool
		env         map[string]string
		unsetEnv    []string
		output      string
	}{
		{
			name: "HOME/USERPROFILE specified",
			env: map[string]string{
				"SDPCTL_CONFIG_DIR": "",
				"XDG_CONFIG_HOME":   filepath.Join(tempDir, ".config"),
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
		{
			name:     "no XDG_CONFIG_HOME specified",
			unsetEnv: []string{"HOME", "XDG_CONFIG_HOME"},
			env: map[string]string{
				"HOME": tempDir,
			},
			output: filepath.Join(tempDir, ".config", "sdpctl"),
		},
	}

	for _, tt := range tests {
		if tt.onlyWindows && runtime.GOOS != "windows" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range tt.unsetEnv {
				old := os.Getenv(v)
				os.Unsetenv(v)
				defer os.Setenv(v, old)
			}
			if tt.env != nil {
				for k, v := range tt.env {
					old := os.Getenv(k)
					os.Setenv(k, v)
					defer os.Setenv(k, old)
				}
			}
			xdg.Reload() // reload env after setting testing variables

			if dir := ConfigDir(); dir != tt.output {
				t.Errorf("Got %s, expected %s", dir, tt.output)
			}
		})
	}
}

func TestAbsolutePath(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "absolute path",
			arg:  "/tmp/path",
			want: "/tmp/path",
		},
		{
			name: "relative path",
			arg:  "tmp/path",
			want: os.ExpandEnv("${PWD}/tmp/path"),
		},
		{
			name: "path with env variable",
			arg:  "${HOME}/tmp/path",
			want: os.ExpandEnv("${HOME}/tmp/path"),
		},
		{
			name: "path with tilde",
			arg:  "~/tmp/path",
			want: os.ExpandEnv("${HOME}/tmp/path"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AbsolutePath(tt.arg)
			if got != tt.want {
				t.Errorf("AbsolutePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
