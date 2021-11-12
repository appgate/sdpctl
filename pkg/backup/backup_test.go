package backup_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/appgate/appgatectl/cmd/appliance/backup"
	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/spf13/cobra"
)

func TestBackupCmd(t *testing.T) {
	applianceUUID := "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
	backupUUID := "fd5ea380-496b-41eb-8bc8-2c84eb36b605"
	registry := httpmock.NewRegistry()

	// Appliance list route
	registry.Register(
		"/appliances",
		httpmock.FileResponse("./fixtures/appliance_list.json"),
	)

    // Initiate backup request
	registry.Register(
		fmt.Sprintf("/appliances/%s/backup", applianceUUID),
		httpmock.FileResponse("./fixtures/appliance_backup_initiated.json"),
	)

	// Backup is processing
	registry.Register(
		fmt.Sprintf("/appliances/%s/backup/%s/status", applianceUUID, backupUUID),
		httpmock.FileResponse("./fixtures/appliance_backup_status_processing.json"),
	)
	defer registry.Teardown()
	registry.Serve()

	stdout := &bytes.Buffer{}
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
		},
		IOOutWriter: stdout,
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

    cmd := backup.NewCmdBackup(f)
    cmdout := make(chan *cobra.Command)
    errors := make(chan error)
    go func() {
        outcmd, err := cmd.ExecuteC()
        if err != nil {
            errors <- err
        }
        cmdout <- outcmd
    }()
    time.Sleep(3 * time.Second)

	registry.Register(
		fmt.Sprintf("/appliances/%s/backup/%s/status", applianceUUID, backupUUID),
		httpmock.FileResponse("./fixtures/appliance_backup_status_done.json"),
	)

    select {
    case e := <-errors:
        t.Fatal(e)
    case o := <-cmdout:
        t.Log(o)
    }
}
