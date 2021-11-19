package configure

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
)

func TestCredentialsCmd(t *testing.T) {
	stdout := new(bytes.Buffer)
	stdin := new(bytes.Buffer)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
		},
		IOOutWriter: stdout,
		Stdin:       stdin,
		StdErr:      stdout,
	}

	tests := map[string]struct {
		env     map[string]string
		args    []string
		wantErr bool
		output  *regexp.Regexp
	}{
		"should fail on no credentials": {
			wantErr: true,
			output:  regexp.MustCompile(`invalid credentials`),
		},
		"should set only username": {
			output: regexp.MustCompile(`Stored credentials in`),
			env: map[string]string{
				"APPGATECTL_USERNAME": "testuser",
			},
		},
		"should set only password": {
			output: regexp.MustCompile("Stored credentials in"),
			env: map[string]string{
				"APPGATECTL_PASSWORD": "testpassword",
			},
		},
		"should set username and password": {
			output: regexp.MustCompile("Stored credentials in"),
			env: map[string]string{
				"APPGATECTL_USERNAME": "testuser",
				"APPGATECTL_PASSWORD": "testpassword",
			},
		},
		"should set credentials from custom envs": {
			output: regexp.MustCompile("Stored credentials in"),
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
			output: regexp.MustCompile(`Stored credentials in.+customcredentialsfile$`),
			args: []string{
				"--file=customcredentialsfile",
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
			}
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
						t.Fatalf("EXPECTED: %+v,\nGOT %+v", data.output.String(), got)
					}
				}
			} else {
				got, readErr := io.ReadAll(stdout)
				if readErr != nil {
					t.Fatal("Error reading output buffer: ", readErr)
				}

				if !data.output.Match(got) {
					t.Fatalf("EXPECTED: %+v,\nGOT: %+v", data.output.String(), got)
				}
			}
		})
	}

}
