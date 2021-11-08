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

// GetAllAppliances from the appgate sdp collective, without any filter.
func GetAllAppliances(ctx context.Context, client *openapi.APIClient, token string) ([]openapi.Appliance, error) {
	appliances, _, err := client.AppliancesApi.AppliancesGet(ctx).OrderBy("name").Authorization(token).Execute()
	if err != nil {
		return nil, err
	}
	return appliances.GetData(), nil
}

// FindPrimaryController The given hostname should match one of the controller's actual admin hostname.
// Hostnames should be compared in a case insensitive way.
func FindPrimaryController(appliances []openapi.Appliance, hostname string) (*openapi.Appliance, error) {

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

func GetApplianceUpgradeStatus(ctx context.Context, client *openapi.APIClient, token, applianceID string) (openapi.InlineResponse2006, error) {
	status, _, err := client.ApplianceUpgradeApi.AppliancesIdUpgradeGet(ctx, applianceID).Authorization(token).Execute()
	if err != nil {
		return status, err
	}
	return status, nil
}

func GetApplianceStats(ctx context.Context, client *openapi.APIClient, token string) (openapi.StatsAppliancesList, *http.Response, error) {
	status, response, err := client.ApplianceStatsApi.StatsAppliancesGet(ctx).Authorization(token).Execute()
	if err != nil {
		return status, response, err
	}
	return status, response, nil
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
