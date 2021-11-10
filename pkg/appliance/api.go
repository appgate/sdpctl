package appliance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	util "github.com/appgate/appgatectl/internal"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

// Appliance is a wrapper aroudn the APIClient for common functions around the appliance API that
// will be used within several commands.
type Appliance struct {
	APIClient  *openapi.APIClient
	HTTPClient *http.Client
	Token      string
}

// GetAll from the appgate sdp collective, without any filter.
func (a *Appliance) GetAll(ctx context.Context) ([]openapi.Appliance, error) {
	appliances, _, err := a.APIClient.AppliancesApi.AppliancesGet(ctx).OrderBy("name").Authorization(a.Token).Execute()
	if err != nil {
		return nil, err
	}
	return appliances.GetData(), nil
}

// FindPrimaryController The given hostname should match one of the controller's actual admin hostname.
// Hostnames should be compared in a case insensitive way.
func (a *Appliance) FindPrimaryController(appliances []openapi.Appliance, hostname string) (*openapi.Appliance, error) {

	controllers := make([]openapi.Appliance, 0)
	type details struct {
		ID        string
		Hostnames []string
		Appliance openapi.Appliance
	}
	data := make(map[string]details)
	for _, a := range appliances {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			controllers = append(controllers, a)
		}
	}
	for _, controller := range controllers {
		var hostnames []string
		hostnames = append(hostnames, strings.ToLower(controller.GetPeerInterface().Hostname))
		if v, ok := controller.GetAdminInterfaceOk(); ok {
			hostnames = append(hostnames, strings.ToLower(v.GetHostname()))
		}
		data[controller.GetId()] = details{
			ID:        controller.GetId(),
			Hostnames: hostnames,
			Appliance: controller,
		}
	}
	count := 0
	var candidate *openapi.Appliance
	for _, c := range data {
		if util.InSlice(strings.ToLower(hostname), c.Hostnames) {
			count++
			candidate = &c.Appliance
		}
	}
	if count > 1 {
		return nil, fmt.Errorf(
			"The given Controller hostname %s is used by more than one appliance."+
				"A unique Controller admin (or peer) hostname is required to perform the upgrade.",
			hostname,
		)
	}
	if candidate != nil {
		return candidate, nil
	}
	return nil, fmt.Errorf(
		"Unable to match the given Controller hostname %q with the actual Controller admin (or peer) hostname",
		hostname,
	)
}

func (a *Appliance) UpgradeStatus(ctx context.Context, applianceID string) (openapi.InlineResponse2006, error) {
	status, _, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradeGet(ctx, applianceID).Authorization(a.Token).Execute()
	if err != nil {
		return status, err
	}
	return status, nil
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
func (a *Appliance) UploadFile(ctx context.Context, f *os.File) error {
	// TODO; replace with custom HTTP client and use application/octet-stream so we can keep track of the progress.
	// and provide the user with feedback of the upload.
	r, err := a.APIClient.ApplianceUpgradeApi.FilesPut(ctx).Authorization(a.Token).File(f).Execute()
	if err != nil {
		if r == nil {
			return fmt.Errorf("no response during upload %w", err)
		}
		if r.StatusCode == http.StatusConflict {
			return fmt.Errorf("%q: already exists %w", f.Name(), err)
		}
		return err
	}
	return nil
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
	}
	_, r, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradePreparePost(ctx, id).ApplianceUpgrade(u).Authorization(a.Token).Execute()
	if err != nil {
		if r == nil {
			return fmt.Errorf("No resposne during prepare %w", err)
		}
		if r.StatusCode == http.StatusConflict {
			return fmt.Errorf("Upgrade in progress on %s %w", id, err)
		}
		return err
	}
	return nil
}
