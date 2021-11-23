package configuration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Test_ConfigDir(t *testing.T) {
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
				"APPGATECTL_CONFIG_DIR": "",
				"XDG_CONFIG_HOME":       "",
				"AppData":               "",
				"HOME":                  tempDir,
			},
			output: filepath.Join(tempDir, ".config", "appgatectl"),
		},
		{
			name: "APPGATECTL_CONFIG_DIR specified",
			env: map[string]string{
				"APPGATECTL_CONFIG_DIR": filepath.Join(tempDir, "appgatectl_dir"),
			},
			output: filepath.Join(tempDir, "appgatectl_dir"),
		},
		{
			name: "XDG_CONFIG_HOME specified",
			env: map[string]string{
				"XDG_CONFIG_HOME": tempDir,
			},
			output: filepath.Join(tempDir, "appgatectl"),
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

func TestLoadCredentialsFile(t *testing.T) {
	tempDir := os.TempDir()

	tests := []struct {
		name        string
		fileName    string
		fileContent string
		fileMode    int
		output      string
		comparison  *Credentials
		wantErr     bool
	}{
		{
			name:        "should fail on invalid credentials",
			fileName:    "credentials",
			fileContent: "username=\npassword=",
			fileMode:    0600,
			output:      "invalid credentials",
			comparison:  &Credentials{},
			wantErr:     true,
		},
		{
			name:        "should fail on invalid mode set",
			fileName:    "credentials",
			fileContent: "username=test\npassword=password",
			fileMode:    0755,
			output:      "invalid permissions on credentials file",
			comparison:  &Credentials{},
			wantErr:     true,
		},
		{
			name:        "should fail on no credentials file set",
			fileName:    "",
			fileContent: "",
			fileMode:    0700,
			output:      "no credentials file set",
			comparison:  &Credentials{},
			wantErr:     true,
		},
		{
			name:        "should pass",
			fileName:    "credentials",
			fileContent: "username=testuser\npassword=password",
			fileMode:    0600,
			output:      "",
			comparison: &Credentials{
				Username: "testuser",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name:        "should pass with only one of the credentials set",
			fileName:    "credentials",
			fileContent: "username=testuser",
			fileMode:    0600,
			output:      "",
			comparison: &Credentials{
				Username: "testuser",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &Config{}
			if len(tt.fileName) > 0 {
				file, _ := os.CreateTemp(tempDir, tt.fileName)
				file.Chmod(os.FileMode(tt.fileMode))
				file.WriteString(tt.fileContent)
				conf.CredentialsFile = file.Name()
				defer file.Close()
			}

			res, err := conf.LoadCredentials()
			if err != nil && tt.wantErr {
				if tt.output != err.Error() {
					t.Fatalf("EXPECTED: %s\n, GOT: %+v", tt.output, err.Error())
				}
			}

			if res != nil {
				if res.Password != tt.comparison.Password || res.Username != tt.comparison.Username {
					t.Fatalf("EXPECTED: %+v,\nGOT: %+v", tt.comparison, res)
				}
			}
		})
	}
}
