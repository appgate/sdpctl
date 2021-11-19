package configure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/viper"
)

func TestCredentialsCmd(t *testing.T) {
	tempPath := filepath.FromSlash(fmt.Sprintf("%s/appgate-testing", os.TempDir()))
	config := &configuration.Config{
		Debug: false,
	}
	cfg, _ := json.Marshal(config)
	cfgPath := fmt.Sprintf("%s/config.json", tempPath)
	os.WriteFile(cfgPath, []byte(cfg), 0700)
	defer os.Remove(cfgPath)
	viper.SetConfigFile(cfgPath)

	stdout := new(bytes.Buffer)
	f := &factory.Factory{
		Config:      config,
		IOOutWriter: stdout,
		StdErr:      stdout,
	}

	tests := map[string]struct {
		file    string
		env     map[string]string
		args    []string
		compare *configuration.Credentials
		wantErr bool
		output  *regexp.Regexp
	}{
		"should fail on no credentials": {
			wantErr: true,
			output:  regexp.MustCompile(`invalid credentials`),
		},
		"should set only username": {
			output:  regexp.MustCompile(`Stored credentials in`),
			compare: &configuration.Credentials{Username: "testuser"},
			env: map[string]string{
				"APPGATECTL_USERNAME": "testuser",
			},
		},
		"should set only password": {
			output:  regexp.MustCompile("Stored credentials in"),
			compare: &configuration.Credentials{Password: "testpassword"},
			env: map[string]string{
				"APPGATECTL_PASSWORD": "testpassword",
			},
		},
		"should set username and password": {
			output:  regexp.MustCompile("Stored credentials in"),
			compare: &configuration.Credentials{Username: "testuser", Password: "testpassword"},
			env: map[string]string{
				"APPGATECTL_USERNAME": "testuser",
				"APPGATECTL_PASSWORD": "testpassword",
			},
		},
		"should set credentials from custom envs": {
			output:  regexp.MustCompile("Stored credentials in"),
			compare: &configuration.Credentials{Username: "testuser", Password: "testpassword"},
			args: []string{
				"--username-env=TEST_USERNAME_ENV",
				"--password-env=TEST_PASSWORD_ENV",
			},
			env: map[string]string{
				"TEST_USERNAME_ENV": "testuser",
				"TEST_PASSWORD_ENV": "testpassword",
			},
		},
		"should set credentials in custom credentials file": {
			file:    "/tmp/appgate-testing/customcredentialsfile",
			output:  regexp.MustCompile(`Stored credentials in.+customcredentialsfile`),
			compare: &configuration.Credentials{Username: "testuser", Password: "testpassword"},
			args: []string{
				"--file=/tmp/appgate-testing/customcredentialsfile",
			},
			env: map[string]string{
				"APPGATECTL_USERNAME": "testuser",
				"APPGATECTL_PASSWORD": "testpassword",
			},
		},
	}

	for test, data := range tests {
		t.Run(test, func(t *testing.T) {
			for key, value := range data.env {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			if len(data.file) <= 0 {
				defaultPath := fmt.Sprintf("/tmp/appgate-testing/credentials%d", time.Now().UnixNano())
				data.file = defaultPath
				data.args = append(data.args, fmt.Sprintf("--file=%s", defaultPath))
			}
			defer os.Remove(data.file)
			cmd := NewCredentialsCmd(f)

			cmd.SetArgs(data.args)
			_, err := cmd.ExecuteC()
			if err != nil {
				if !data.wantErr {
					t.Fatalf("EXPECTED: %+v,\nGOT: %+v", data.output, err)
				} else {
					got, readErr := io.ReadAll(stdout)
					if readErr != nil {
						t.Fatal("Error reading output buffer: ", readErr)
					}
					if !data.output.Match(got) {
						t.Fatalf("EXPECTED: %+v,\nGOT %+v", data.output.String(), string(got))
					}
				}
			} else {
				got, readErr := io.ReadAll(stdout)
				if readErr != nil {
					t.Fatal("Error reading output buffer: ", readErr)
				}

				if !data.output.Match(got) {
					t.Fatalf("EXPECTED: %+v,\nGOT: %+v", data.output.String(), string(got))
				}

				viper.ReadInConfig()
				storedCredsPath := viper.GetString("credentials_file")
				if storedCredsPath != data.file {
					t.Fatalf("EXPECTED: %+v,\nGOT: %+v", data.file, storedCredsPath)
				}
				config.CredentialsFile = storedCredsPath

				storedCredentials, err := config.GetCredentialsFromFile()
				if err != nil {
					t.Fatal(err)
				}

				if len(data.compare.Username) > 0 && data.compare.Username != storedCredentials.Username {
					t.Fatalf("EXPECTED: %s,\nGOT: %s", data.compare.Username, storedCredentials.Username)
				}
				if len(data.compare.Password) > 0 && data.compare.Password != storedCredentials.Password {
					t.Fatalf("EXPECTED: %s,\nGOT: %s", data.compare.Password, storedCredentials.Password)
				}
			}
		})
	}

}
