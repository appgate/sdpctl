package backup

import (
	"context"
	"fmt"
	"os"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/api"
)

type Backup struct {
	APIClient *openapi.APIClient
	Token     string
	Version   int
}

func New(c *openapi.APIClient, t string, v int) *Backup {
	return &Backup{
		APIClient: c,
		Token:     t,
		Version:   v,
	}
}

// Initiate an Appliance Backup. The progress can be followed by polling the Appliance via "GET appliances/{id}/backup/{backupId}/status".
func (b *Backup) Initiate(ctx context.Context, applianceID string, logs, audit bool) (string, error) {
	o := openapi.AppliancesIdBackupPostRequest{
		Logs:  &logs,
		Audit: &audit,
	}
	status, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, applianceID).Authorization(b.Token).AppliancesIdBackupPostRequest(o).Execute()
	if err != nil {
		return "", api.HTTPErrorResponse(response, err)
	}

	return status.GetId(), nil
}

// Download a completed Appliance Backup with the given ID of an Appliance
func (b *Backup) Download(ctx context.Context, applianceID, backupID string) (*os.File, error) {
	ctxWithGPGAccept := context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", b.Version))
	file, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdGet(ctxWithGPGAccept, applianceID, backupID).Authorization(b.Token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return *file, nil
}

const (
	// https://github.com/appgate/sdp-api-specification/blob/0cae2de511a135ca1c29beb89fe9d38e83ffc4f1/appliance_backup.yml#L87-L88
	Processing string = "processing"
	Done       string = "done"
)

func (b *Backup) Status(ctx context.Context, applianceID, backupID string) (string, error) {
	status, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdStatusGet(ctx, applianceID, backupID).Authorization(b.Token).Execute()
	if err != nil {
		return "", api.HTTPErrorResponse(response, err)
	}
	return status.GetStatus(), nil
}
