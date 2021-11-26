package appliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/appgate/appgatectl/pkg/api"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

// Appliance is a wrapper aroudn the APIClient for common functions around the appliance API that
// will be used within several commands.
type Appliance struct {
	APIClient           *openapi.APIClient
	HTTPClient          *http.Client
	Token               string
	UpgradeStatusWorker WaitForUpgradeStatus
}

// GetAll from the appgate sdp collective, without any filter.
func (a *Appliance) GetAll(ctx context.Context) ([]openapi.Appliance, error) {
	appliances, _, err := a.APIClient.AppliancesApi.AppliancesGet(ctx).OrderBy("name").Authorization(a.Token).Execute()
	if err != nil {
		return nil, err
	}
	return appliances.GetData(), nil
}

const (
	//lint:file-ignore U1000 All avaliable upgrade statuses
	UpgradeStatusIdle        = "idle"
	UpgradeStatusStarted     = "started"
	UpgradeStatusDownloading = "downloading"
	UpgradeStatusVerifying   = "verifying"
	UpgradeStatusReady       = "ready"
	UpgradeStatusInstalling  = "installing"
	UpgradeStatusSuccess     = "success"
	UpgradeStatusFailed      = "failed"
)

func (a *Appliance) UpgradeStatus(ctx context.Context, applianceID string) (openapi.InlineResponse2006, error) {
	status, _, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradeGet(ctx, applianceID).Authorization(a.Token).Execute()
	if err != nil {
		return status, err
	}
	return status, nil
}

func (a *Appliance) UpgradeCancel(ctx context.Context, applianceID string) error {
	response, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradeDelete(ctx, applianceID).Authorization(a.Token).Execute()
	if err != nil {
		if response != nil && response.StatusCode >= 400 {
			responseBody, errRead := io.ReadAll(response.Body)
			if errRead != nil {
				return err
			}
			errBody := api.GenericErrorResponse{}
			if err := json.Unmarshal(responseBody, &errBody); err != nil {
				return err
			}
			return fmt.Errorf("%s %v", errBody.Message, errBody.Errors)
		}
		return err
	}
	return nil
}

func (a *Appliance) Stats(ctx context.Context) (openapi.StatsAppliancesList, *http.Response, error) {
	status, response, err := a.APIClient.ApplianceStatsApi.StatsAppliancesGet(ctx).Authorization(a.Token).Execute()
	if err != nil {
		return status, response, err
	}
	return status, response, nil
}

var ErrFileNotFound = errors.New("File not found")

// FileStatus Get the status of a File uploaded to the current Controller.
func (a *Appliance) FileStatus(ctx context.Context, filename string) (openapi.File, error) {
	f, r, err := a.APIClient.ApplianceUpgradeApi.FilesFilenameGet(ctx, filename).Authorization(a.Token).Execute()
	if err != nil {
		if r.StatusCode == http.StatusNotFound {
			return f, fmt.Errorf("%q: %w", filename, ErrFileNotFound)
		}
		return f, err
	}
	return f, nil
}

// UploadFile directly to the current Controller. Note that the File is stored only on the current Controller, not synced between Controllers.
func (a *Appliance) UploadFile(ctx context.Context, r io.ReadCloser, headers map[string]string) error {
	httpClient := a.HTTPClient
	cfg := a.APIClient.GetConfig()
	url, err := cfg.ServerURLWithContext(ctx, "ApplianceUpgradeApiService.FilesPut")
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, url+"/files", r)
	if err != nil {
		return err
	}
	for k, v := range cfg.DefaultHeader {
		req.Header.Add(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", a.Token)
	response, err := httpClient.Do(req)
	if err != nil {
		if response == nil {
			return fmt.Errorf("no response during upload %w", err)
		}
		if response.StatusCode == http.StatusConflict {
			return fmt.Errorf("already exists %w", err)
		}
		if response.StatusCode > 400 {
			responseBody, errRead := io.ReadAll(response.Body)
			if errRead != nil {
				return err
			}
			errBody := api.GenericErrorResponse{}
			if err := json.Unmarshal(responseBody, &errBody); err != nil {
				return err
			}
			return fmt.Errorf("%s %v", errBody.Message, errBody.Errors)
		}
		return err
	}
	response.Body.Close()
	return nil
}

func (a *Appliance) ListFiles(ctx context.Context) ([]openapi.File, error) {
	list, _, err := a.APIClient.ApplianceUpgradeApi.FilesGet(ctx).Authorization(a.Token).Execute()
	if err != nil {
		return nil, err
	}
	return list.GetData(), nil
}

// DeleteFile Delete a File from the current Controller.
func (a *Appliance) DeleteFile(ctx context.Context, filename string) error {
	_, err := a.APIClient.ApplianceUpgradeApi.FilesFilenameDelete(ctx, filename).Authorization(a.Token).Execute()
	if err != nil {
		return err
	}
	return nil
}

func (a *Appliance) PrepareFileOn(ctx context.Context, filename, id string) error {
	u := openapi.ApplianceUpgrade{
		ImageUrl: filename,
		// TODO: update use user input args
		// verify version >= 14 ?
		DevKeyring: openapi.PtrBool(true),
	}
	_, r, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradePreparePost(ctx, id).ApplianceUpgrade(u).Authorization(a.Token).Execute()
	if err != nil {
		if r == nil {
			return fmt.Errorf("No response during prepare %w", err)
		}
		if r.StatusCode == http.StatusConflict {
			return fmt.Errorf("Upgrade in progress on %s %w", id, err)
		}
		return err
	}
	return nil
}
