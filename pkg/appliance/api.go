package appliance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

// GetAllAppliances from the appgate sdp collective, without any filter.
func GetAllAppliances(ctx context.Context, client *openapi.APIClient, token string) ([]openapi.Appliance, error) {
	appliances, _, err := client.AppliancesApi.AppliancesGet(ctx).OrderBy("name").Authorization(token).Execute()
	if err != nil {
		return nil, err
	}
	return appliances.GetData(), nil
}

func GetApplianceUpgradeStatus(ctx context.Context, client *openapi.APIClient, token, applianceID string) (openapi.InlineResponse2006, error) {
	status, _, err := client.ApplianceUpgradeApi.AppliancesIdUpgradeGet(ctx, applianceID).Authorization(token).Execute()
	if err != nil {
		return status, err
	}
	return status, nil
}

var ErrFileNotFound = errors.New("File not found")

// GetFileStatus Get the status of a File uploaded to the current Controller.
func GetFileStatus(ctx context.Context, client *openapi.APIClient, token, filename string) (openapi.File, error) {
	f, r, err := client.ApplianceUpgradeApi.FilesFilenameGet(ctx, filename).Authorization(token).Execute()
	if err != nil {
		if r.StatusCode == http.StatusNotFound {
			return f, fmt.Errorf("%q: %w", filename, ErrFileNotFound)
		}
		return f, err
	}
	return f, nil
}

// UploadFile directly to the current Controller. Note that the File is stored only on the current Controller, not synced between Controllers.
func UploadFile(ctx context.Context, client *openapi.APIClient, token string, f *os.File) error {
	// TODO; replace with custom HTTP client and use application/octet-stream so we can keep track of the progress.
	// and provide the user with feedback of the upload.
	r, err := client.ApplianceUpgradeApi.FilesPut(ctx).Authorization(token).File(f).Execute()
	if err != nil {
		if r.StatusCode == http.StatusConflict {
			return fmt.Errorf("%q: already exists %w", f.Name(), err)
		}
		return err
	}
	return nil
}

// DeleteFile Delete a File from the current Controller.
func DeleteFile(ctx context.Context, client *openapi.APIClient, token, filename string) error {
	_, err := client.ApplianceUpgradeApi.FilesFilenameDelete(ctx, filename).Authorization(token).Execute()
	if err != nil {
		return err
	}
	return nil
}
