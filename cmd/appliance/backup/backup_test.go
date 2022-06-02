package backup

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
)

func TestBackupCmd(t *testing.T) {
	applianceUUID := "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
	backupUUID := "fd5ea380-496b-41eb-8bc8-2c84eb36b605"
	registry := httpmock.NewRegistry()

	// Appliance list route
	registry.Register(
		"/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
	)
	// Appliance stats route
	registry.Register(
		"/stats/appliances",
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
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_backup_status_done.json"),
	)
	// Download backup
	registry.Register(
		fmt.Sprintf("/appliances/%s/backup/%s", applianceUUID, backupUUID),
		httpmock.FileResponse(),
	)
	defer registry.Teardown()
	registry.Serve()

	buf := new(bytes.Buffer)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://appgate.com:%d", registry.Port),
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

func TestBackupCmdDisabledAPI(t *testing.T) {
	registry := httpmock.NewRegistry()

	// Appliance list route
	registry.Register(
		"/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
	)
	// Appliance stats route
	registry.Register(
		"/stats/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
	)
	// Backup state
	registry.Register(
		"/global-settings",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_global_options_backup_disabled.json"),
	)
	defer registry.Teardown()
	registry.Serve()

	buf := new(bytes.Buffer)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://appgate.com:%d", registry.Port),
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
	cmd.SetArgs([]string{"--destination=/tmp/appgate-testing", "--no-interactive"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	reg := regexp.MustCompile(`Using '--no-interactive' flag while backup API is disabled. Use the 'sdpctl appliance backup api' command to enable it before trying again.`)
	_, err := cmd.ExecuteC()
	if err != nil {
		if !reg.MatchString(err.Error()) {
			t.Fatalf("Error message did not match expected.\nWANT: %s\nGOT: %s", reg.String(), err.Error())
		}
	}
}

func TestBackupCmdNoState(t *testing.T) {
	registry := httpmock.NewRegistry()

	// Appliance list route
	registry.Register(
		"/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
	)
	// Appliance stats route
	registry.Register(
		"/stats/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
	)
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
