package backup

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
)

func TestBackupCmd(t *testing.T) {
	applianceUUID := "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
	backupUUID := "fd5ea380-496b-41eb-8bc8-2c84eb36b605"
	registry := httpmock.NewRegistry(t)

	// Appliance list route
	registry.Register(
		"/admin/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
	)
	// Appliance stats route
	registry.Register(
		"/admin/appliances/status",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
	)
	// Backup state
	registry.Register(
		"/admin/global-settings",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_global_options.json"),
	)
	// Initiate backup request
	registry.Register(
		fmt.Sprintf("/admin/appliances/%s/backup", applianceUUID),
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_backup_initiated.json"),
	)
	// Backup is done
	registry.Register(
		fmt.Sprintf("/admin/appliances/%s/backup/%s/status", applianceUUID, backupUUID),
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_backup_status_done.json"),
	)
	// Download backup
	registry.Register(
		fmt.Sprintf("/admin/appliances/%s/backup/%s", applianceUUID, backupUUID),
		httpmock.FileResponse(),
	)
	defer registry.Teardown()
	registry.Serve()

	buf := new(bytes.Buffer)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://appgate.test:%d", registry.Port),
		},
		IOOutWriter: buf,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.Appliance = func(c *configuration.Config) (*appliance.Appliance, error) {
		api, _ := f.APIClient(c)

		a := &appliance.Appliance{
			APIClient:  api,
			HTTPClient: api.GetConfig().HTTPClient,
			Token:      "",
		}
		return a, nil
	}

	cmd := NewCmdBackup(f)
	cmd.Flags().Bool("no-interactive", false, "usage")
	cmd.Flags().Bool("ci-mode", false, "ci-mode")
	cmd.SetArgs([]string{"--destination=/tmp/appgate-testing", "--primary", "--no-interactive"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}
	got, err := io.ReadAll(buf)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	reg := regexp.MustCompile(`Wrote backup file.+file=/tmp/appgate-testing/appgate_backup_.+.bkp`)
	if res := reg.Find(got); res == nil {
		t.Fatalf("result matching failed. WANT: %+v, GOT: %+v", reg.String(), string(got))
	}
}

func TestBackupCmdFailedOnServer(t *testing.T) {
	applianceUUID := "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
	backupUUID := "fd5ea380-496b-41eb-8bc8-2c84eb36b605"
	registry := httpmock.NewRegistry(t)

	// Appliance list route
	registry.Register(
		"/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
	)
	// Appliance stats route
	registry.Register(
		"/appliances/status",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
	)
	// Backup state
	registry.Register(
		"/global-settings",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_global_options.json"),
	)
	// Initiate backup request
	registry.Register(
		fmt.Sprintf("/appliances/%s/backup", applianceUUID),
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_backup_initiated.json"),
	)
	// Backup is done
	registry.Register(
		fmt.Sprintf("/appliances/%s/backup/%s/status", applianceUUID, backupUUID),
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_backup_status_failed.json"),
	)
	defer registry.Teardown()
	registry.Serve()

	buf := new(bytes.Buffer)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://appgate.test:%d", registry.Port),
		},
		IOOutWriter: buf,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.Appliance = func(c *configuration.Config) (*appliance.Appliance, error) {
		api, _ := f.APIClient(c)

		a := &appliance.Appliance{
			APIClient:  api,
			HTTPClient: api.GetConfig().HTTPClient,
			Token:      "",
		}
		return a, nil
	}

	cmd := NewCmdBackup(f)
	cmd.Flags().Bool("no-interactive", false, "usage")
	cmd.Flags().Bool("ci-mode", false, "ci-mode")
	cmd.SetArgs([]string{"--destination=/tmp/appgate-testing", "--primary", "--no-interactive"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := cmd.ExecuteC()
	if err == nil {
		reg := regexp.MustCompile(`could not backup the Controller: something went wrong`)
		if res := reg.MatchString(err.Error()); !res {
			t.Fatalf("result matching failed. WANT: %+v, GOT: %+v", reg.String(), string(err.Error()))
		}
	}
}

func TestBackupCmdDisabledAPI(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	// Backup state
	registry.Register(
		"/admin/global-settings",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_global_options_backup_disabled.json"),
	)
	defer registry.Teardown()
	registry.Serve()

	buf := new(bytes.Buffer)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://appgate.test:%d", registry.Port),
		},
		IOOutWriter: buf,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.Appliance = func(c *configuration.Config) (*appliance.Appliance, error) {
		api, _ := f.APIClient(c)

		a := &appliance.Appliance{
			APIClient:  api,
			HTTPClient: api.GetConfig().HTTPClient,
			Token:      "",
		}
		return a, nil
	}

	cmd := NewCmdBackup(f)
	cmd.Flags().Bool("no-interactive", false, "usage")
	cmd.Flags().Bool("ci-mode", false, "ci-mode")
	cmd.SetArgs([]string{"--destination=/tmp/appgate-testing", "--no-interactive"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	reg := regexp.MustCompile(`Using '--no-interactive' flag while Backup API is disabled. Use the 'sdpctl appliance backup api' command to enable it before trying again`)
	_, err := cmd.ExecuteC()
	if err != nil {
		if !reg.MatchString(err.Error()) {
			t.Fatalf("Error message did not match expected.\nWANT: %s\nGOT: %s", reg.String(), err.Error())
		}
	}
}

func TestBackupCmdNoState(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	defer registry.Teardown()
	registry.Serve()

	buf := new(bytes.Buffer)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
		},
		IOOutWriter: buf,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.Appliance = func(c *configuration.Config) (*appliance.Appliance, error) {
		api, _ := f.APIClient(c)

		a := &appliance.Appliance{
			APIClient:  api,
			HTTPClient: api.GetConfig().HTTPClient,
			Token:      "",
		}
		return a, nil
	}

	cmd := NewCmdBackup(f)
	cmd.Flags().Bool("no-interactive", false, "usage")
	cmd.Flags().Bool("ci-mode", false, "ci-mode")
	cmd.SetArgs([]string{"--destination=/tmp/appgate-testing", "--no-interactive"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	reg := regexp.MustCompile(`Backup failed due to error while --no-interactive flag is set`)
	_, err := cmd.ExecuteC()
	if err != nil {
		if !reg.MatchString(err.Error()) {
			t.Fatalf("Error message did not match expected.\nWANT: %s\nGOT: %s", reg.String(), err.Error())
		}
	}
}

func TestBackupCmdDirectValidation(t *testing.T) {
	// Test direct validation of GetBackupPassphrase function with stdin
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid passphrase",
			input:   "ValidP@ssw0rd123\n",
			wantErr: false,
		},
		{
			name:    "invalid passphrase with space",
			input:   "invalid passphrase\n",
			wantErr: true,
			errMsg:  prompt.PassphraseInvalidMessage,
		},
		{
			name:    "invalid passphrase with emoji",
			input:   "password😀\n",
			wantErr: true,
			errMsg:  prompt.PassphraseInvalidMessage,
		},
		{
			name:    "invalid passphrase with tab",
			input:   "pass\tword\n",
			wantErr: true,
			errMsg:  prompt.PassphraseInvalidMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := bytes.NewBufferString(tt.input)
			// Test with hasStdin=true to bypass the TTY check
			result, err := prompt.GetBackupPassphrase(stdin, false, true, "test message")

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBackupPassphrase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("GetBackupPassphrase() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				return
			}

			if !tt.wantErr && result != strings.TrimSuffix(tt.input, "\n") {
				t.Errorf("GetBackupPassphrase() = %v, want %v", result, strings.TrimSuffix(tt.input, "\n"))
			}
		})
	}
}
