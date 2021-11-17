package backup

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

func TestBackupCmd(t *testing.T) {
	applianceUUID := "4c07bc67-57ea-42dd-b702-c2d6c45419fc"
	backupUUID := "fd5ea380-496b-41eb-8bc8-2c84eb36b605"
	registry := httpmock.NewRegistry()

	// Appliance list route
	registry.Register(
		"/appliances",
		httpmock.JSONResponse("./fixtures/appliance_list.json"),
	)
	// Initiate backup request
	registry.Register(
		fmt.Sprintf("/appliances/%s/backup", applianceUUID),
		httpmock.JSONResponse("./fixtures/appliance_backup_initiated.json"),
	)
	// Backup is done
	registry.Register(
		fmt.Sprintf("/appliances/%s/backup/%s/status", applianceUUID, backupUUID),
		httpmock.JSONResponse("./fixtures/appliance_backup_status_done.json"),
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
	cmd.SetArgs([]string{"--destination=/tmp/appgate-testing"})
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
	reg := regexp.MustCompile(`wrote backup file to '/tmp/appgate-testing/appgate_backup_.+.bkp`)
	if res := reg.Find(got); res == nil {
		t.Fatalf("result matching failed.")
	}
}
