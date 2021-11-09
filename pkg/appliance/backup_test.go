package appliance_test

import (
	"testing"

	"github.com/appgate/appgatectl/pkg/httpmock"
)

func TestBackupCmd(t *testing.T) {
    registry := httpmock.NewRegistry()
    registry.Register("/admin/backup/", httpmock.FileResponse("../../appliance/fixures/appliance_backup.json"))
}
