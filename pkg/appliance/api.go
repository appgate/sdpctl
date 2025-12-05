package appliance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Appliance is a wrapper around the APIClient for common functions around the appliance API that
// will be used within several commands.
type Appliance struct {
	APIClient           *openapi.APIClient
	HTTPClient          *http.Client
	Token               string
	UpgradeStatusWorker WaitForUpgradeStatus
	ApplianceStats      WaitForApplianceStatus
}

// List from the Collective
// Filter is applied in app after getting all the appliances because the auto generated API screws up the 'filterBy' command
func (a *Appliance) List(ctx context.Context, filter map[string]map[string]string, orderBy []string, descending bool) ([]openapi.Appliance, error) {
	appliances, response, err := a.APIClient.AppliancesApi.AppliancesGet(ctx).OrderBy("name").Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	result, _, err := FilterAppliances(appliances.GetData(), filter, orderBy, descending)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Get return a single appliance based on applianceID
func (a *Appliance) Get(ctx context.Context, applianceID string) (*openapi.Appliance, error) {
	appliance, response, err := a.APIClient.AppliancesApi.AppliancesIdGet(ctx, applianceID).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return appliance, nil
}

const (
	//lint:file-ignore U1000 All available upgrade statuses
	UpgradeStatusIdle        = "idle"
	UpgradeStatusStarted     = "started"
	UpgradeStatusDownloading = "downloading"
	UpgradeStatusVerifying   = "verifying"
	UpgradeStatusReady       = "ready"
	UpgradeStatusInstalling  = "installing"
	UpgradeStatusSuccess     = "success"
	UpgradeStatusFailed      = "failed"
	fileInProgress           = "InProgress"
	FileReady                = "Ready"
	FileFailed               = "Failed"
)

func (a *Appliance) UpgradeStatus(ctx context.Context, applianceID string) (*openapi.AppliancesIdUpgradeDelete200Response, error) {
	status, response, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradeGet(ctx, applianceID).Execute()
	if err != nil {
		return status, api.HTTPErrorResponse(response, err)
	}
	return status, nil
}

func (a *Appliance) UpgradeStatusRetry(ctx context.Context, applianceID string) (*openapi.AppliancesIdUpgradeDelete200Response, error) {
	var status *openapi.AppliancesIdUpgradeDelete200Response
	err := backoff.Retry(func() error {
		s, err := a.UpgradeStatus(ctx, applianceID)
		if err != nil {
			return err
		}
		status = s
		return nil
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
	return status, err
}

type UpgradeStatusResult struct {
	Status, Details, Name string
}

// UpgradeStatusMap return a map with appliance.id, UpgradeStatusResult
func (a *Appliance) UpgradeStatusMap(ctx context.Context, appliances []openapi.Appliance) (map[string]UpgradeStatusResult, error) {
	type result struct {
		id, status, details, name string
	}
	g, ctx := errgroup.WithContext(ctx)
	c := make(chan result)
	for _, appliance := range appliances {
		i := appliance
		g.Go(func() error {
			status, err := a.UpgradeStatusRetry(ctx, i.GetId())
			if err != nil {
				return fmt.Errorf("Could not read status of %s %w", i.GetId(), err)
			}
			select {
			case c <- result{
				id:      i.GetId(),
				status:  status.GetStatus(),
				details: status.GetDetails(),
				name:    i.GetName(),
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}
	go func() {
		g.Wait()
		close(c)
	}()
	m := make(map[string]UpgradeStatusResult)
	for r := range c {
		m[r.id] = UpgradeStatusResult{
			Status:  r.status,
			Details: r.details,
			Name:    r.name,
		}
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return m, nil
}

func (a *Appliance) UpgradeCancel(ctx context.Context, applianceID string) error {
	_, response, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradeDelete(ctx, applianceID).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	return nil
}

func isEnabled(status *string) bool {
	return *status != "n/a"
}

func GetActiveFunctionsDeprecated(appliance openapi.StatsAppliancesListAllOfData) []openapi.ApplianceFunction {
	functions := []openapi.ApplianceFunction{}

	if v, ok := appliance.GetControllerOk(); ok && isEnabled(openapi.PtrString(v.GetStatus())) {
		functions = append(functions, FunctionController)
	}
	if v, ok := appliance.GetGatewayOk(); ok && isEnabled(openapi.PtrString(v.GetStatus())) {
		functions = append(functions, FunctionGateway)
	}
	if v, ok := appliance.GetPortalOk(); ok && isEnabled(openapi.PtrString(v.GetStatus())) {
		functions = append(functions, FunctionPortal)
	}
	if v, ok := appliance.GetConnectorOk(); ok && isEnabled(openapi.PtrString(v.GetStatus())) {
		functions = append(functions, FunctionConnector)
	}
	if v, ok := appliance.GetLogServerOk(); ok && isEnabled(openapi.PtrString(v.GetStatus())) {
		functions = append(functions, FunctionLogServer)
	}
	if v, ok := appliance.GetLogForwarderOk(); ok && isEnabled(openapi.PtrString(v.GetStatus())) {
		functions = append(functions, FunctionLogForwarder)
	}

	return functions
}

func translateDeprecatedStatus(toTranslate *openapi.StatsAppliancesList) *openapi.ApplianceWithStatusList {
	translatedResponse := openapi.ApplianceWithStatusList{
		Data:       []openapi.ApplianceWithStatus{},
		Range:      toTranslate.Range,
		OrderBy:    toTranslate.OrderBy,
		Descending: toTranslate.Descending,
		Queries:    toTranslate.Queries,
		TotalCount: openapi.PtrInt32(int32(len(toTranslate.Data))),
	}
	for _, status := range toTranslate.Data {
		newStatus := openapi.ApplianceWithStatus{
			Id:               status.Id,
			Name:             *status.Name,
			Tags:             status.Tags,
			Site:             status.SiteName,
			Status:           status.Status,
			State:            status.State,
			ApplianceVersion: status.Version,
			Controller: &openapi.ApplianceAllOfController{
				Enabled: openapi.PtrBool(isEnabled(status.Controller.Status)),
			},
			LogServer: &openapi.ApplianceAllOfLogServer{
				Enabled: openapi.PtrBool(isEnabled(status.LogServer.Status)),
			},
			LogForwarder: &openapi.ApplianceAllOfLogForwarder{
				Enabled: openapi.PtrBool(isEnabled(status.LogForwarder.Status)),
			},
			Gateway: &openapi.ApplianceAllOfGateway{
				Enabled: openapi.PtrBool(isEnabled(status.Gateway.Status)),
			},
			Connector: &openapi.ApplianceAllOfConnector{
				Enabled: openapi.PtrBool(isEnabled(status.Connector.Status)),
			},
			Portal: &openapi.Portal{
				Enabled: openapi.PtrBool(isEnabled(status.Portal.Status)),
			},
			Functions: GetActiveFunctionsDeprecated(status),
			Cpu:       status.Cpu,
			Memory:    status.Memory,
			Disk:      status.Disk,
			Details: &openapi.ApplianceWithStatusAllOfDetails{
				Version:      status.Version,
				VolumeNumber: openapi.PtrInt32(int32(*status.VolumeNumber)),
				Cpu: &openapi.SystemInfo{
					Percent: status.Cpu,
				},
				Memory: &openapi.SystemInfo{
					Percent: status.Memory,
				},
				Disk: &openapi.SystemInfo{
					Percent: status.Disk,
				},
				Status: status.Status,
				Roles: &openapi.Roles{
					Controller: &openapi.ControllerRole{
						Status:          status.Controller.Status,
						Details:         status.Controller.Details,
						MaintenanceMode: status.Controller.MaintenanceMode,
						DatabaseSize:    status.Controller.DatabaseSize,
					},
					LogServer: &openapi.ApplianceRole{
						Status:  status.LogServer.Status,
						Details: status.LogServer.Details,
					},
					LogForwarder: &openapi.ApplianceRole{
						Status:  status.LogForwarder.Status,
						Details: status.LogForwarder.Details,
					},
					Gateway: &openapi.ApplianceWithSessionsRole{
						Status:           status.Gateway.Status,
						Details:          status.Gateway.Details,
						NumberOfSessions: status.Gateway.NumberOfSessions,
					},
					Connector: &openapi.ApplianceWithSessionsRole{
						Status:           status.Connector.Status,
						Details:          status.Connector.Details,
						NumberOfSessions: status.Connector.NumberOfSessions,
					},
					Portal: &openapi.ApplianceWithSessionsRole{
						Status:           status.Portal.Status,
						Details:          status.Portal.Details,
						NumberOfSessions: status.Portal.NumberOfSessions,
					},
					Appliance: &openapi.ApplianceBaseRole{
						Status:         status.Appliance.Status,
						Details:        status.Appliance.Details,
						LogDestination: status.Appliance.LogDestination,
					},
				},
			},
		}

		if *status.Status != "offline" {
			newStatus.Details.Network = &openapi.NetworkInfo{
				BusiestNic: status.Network.BusiestNic,
				Details: &map[string]openapi.NetworkInfoDetailsValue{
					*status.Network.BusiestNic: {
						Dropin:  openapi.PtrInt32(int32(*status.Network.Dropin)),
						Dropout: openapi.PtrInt32(int32(*status.Network.Dropout)),
						TxSpeed: status.Network.TxSpeed,
						RxSpeed: status.Network.RxSpeed,
					},
				},
			}
			newStatus.Details.Disk = &openapi.SystemInfo{

				Percent: status.Disk,
			}
			if status.DiskInfo != nil {
				newStatus.Details.Disk.Total = openapi.PtrInt64(int64(*status.DiskInfo.Total))
				newStatus.Details.Disk.Used = openapi.PtrInt64(int64(*status.DiskInfo.Used))
				newStatus.Details.Disk.Free = openapi.PtrInt64(int64(*status.DiskInfo.Free))
			}
			newStatus.Details.Upgrade = &openapi.ApplianceWithStatusAllOfDetailsUpgrade{
				Status:  status.Upgrade.Status,
				Details: status.Upgrade.Details,
			}
		}

		translatedResponse.Data = append(translatedResponse.Data, newStatus)
	}
	return &translatedResponse
}

var warningDisplayed = false

func (a *Appliance) ApplianceStatus(ctx context.Context, filter map[string]map[string]string, orderBy []string, descending bool) (*openapi.ApplianceWithStatusList, *http.Response, error) {
	status, response, err := a.APIClient.AppliancesApi.AppliancesStatusGet(ctx).Execute()
	if response != nil && response.StatusCode == 404 {
		if !warningDisplayed {
			fmt.Fprintln(os.Stderr, "WARNING: Status endpoint not found, falling back to old stats/appliances")
			warningDisplayed = true
		}
		var oldStatus *openapi.StatsAppliancesList
		oldStatus, response, err = a.APIClient.ApplianceStatsDeprecatedApi.StatsAppliancesGet(ctx).Execute()
		if err != nil {
			return nil, response, api.HTTPErrorResponse(response, err)
		}
		status = translateDeprecatedStatus(oldStatus)
	}
	if err != nil {
		return status, response, api.HTTPErrorResponse(response, err)
	}
	stats, _, err := FilterApplianceStats(status.GetData(), filter, orderBy, descending)
	if err != nil {
		return status, response, err
	}
	stats, err = orderApplianceStats(stats, orderBy, descending)
	if err != nil {
		return status, response, err
	}
	status.SetData(stats)
	return status, response, nil
}

// FileStatus Get the status of a File uploaded to the current Controller.
func (a *Appliance) FileStatus(ctx context.Context, filename string) (*openapi.File, error) {
	log := logrus.WithField("file", filename)
	log.Info("checking file status")
	f, r, err := a.APIClient.ApplianceUpgradeApi.FilesFilenameGet(ctx, filename).Execute()
	defer log.WithField("status", f.GetStatus()).Info("got file status")
	if err != nil {
		if r.StatusCode == http.StatusNotFound {
			return f, fmt.Errorf("%q: %w", filename, api.ErrFileNotFound)
		}
		return f, api.HTTPErrorResponse(r, err)
	}
	return f, nil
}

// UploadFile directly to the current Controller. Note that the File is stored only on the current Controller, not synced between Controllers.
func (a *Appliance) UploadFile(ctx context.Context, r io.Reader, headers map[string]string) error {
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

	var filename string
	for k, v := range headers {
		req.Header.Set(k, v)
		rx := regexp.MustCompile(`[\w-_\.]+\.\w+`)
		if match := rx.FindString(v); k == "Content-Disposition" && len(match) > 0 {
			filename = rx.FindString(v)
		}
	}
	log := logrus.WithField("file", filename)
	log.Info("uploading file")

	response, err := httpClient.Do(req)
	if err != nil {
		if response == nil {
			return fmt.Errorf("No response during upload %w", err)
		}
		if response.StatusCode == http.StatusConflict {
			return fmt.Errorf("Already exists %w", err)
		}
		return api.HTTPErrorResponse(response, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return api.HTTPErrorResponse(response, err)
	}
	log.Info("file upload finished")
	return nil
}

func (a *Appliance) UploadToController(ctx context.Context, url, filename string) error {
	response, err := a.APIClient.ApplianceUpgradeApi.FilesPost(ctx).FilesGetRequest1(openapi.FilesGetRequest1{
		Url:      url,
		Filename: filename,
	}).Execute()
	if err != nil {
		if response == nil {
			return fmt.Errorf("No response during upload %w", err)
		}
		if response.StatusCode == http.StatusConflict {
			return fmt.Errorf("Already exists %w", err)
		}
		return api.HTTPErrorResponse(response, err)
	}

	return nil
}

func (a *Appliance) ListFiles(ctx context.Context, orderBy []string, descending bool) ([]openapi.File, error) {
	list, response, err := a.APIClient.ApplianceUpgradeApi.FilesGet(ctx).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return orderApplianceFiles(list.GetData(), orderBy, descending)
}

// DeleteFile Delete a File from the current Controller.
func (a *Appliance) DeleteFile(ctx context.Context, filename string) error {
	log := logrus.WithField("file", filename)
	log.Info("Deleting file from repository")
	response, err := a.APIClient.ApplianceUpgradeApi.FilesFilenameDelete(ctx, filename).Execute()
	if err != nil {
		log.WithError(err).Error("failed to delete file")
		log.WithField("response", response).Debug("got response from server")
		return api.HTTPErrorResponse(response, err)
	}
	log.Info("file deleted")
	return nil
}

func (a *Appliance) PrepareFileOn(ctx context.Context, filename, id string, devKeyring bool) (string, error) {
	u := openapi.ApplianceUpgrade{
		ImageUrl: filename,
	}
	if devKeyring {
		// Only set dev keyring if it is true
		// will prevent errors with older api-version that don't support dev-keyring
		u.DevKeyring = openapi.PtrBool(devKeyring)
	}
	change, r, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradePreparePost(ctx, id).ApplianceUpgrade(u).Execute()
	if err != nil {
		if r == nil {
			return "", fmt.Errorf("No response during prepare %w", err)
		}
		if r.StatusCode == http.StatusConflict {
			return "", fmt.Errorf("Upgrade in progress on %s %w", id, err)
		}
		return "", api.HTTPErrorResponse(r, err)
	}

	return change.GetId(), nil
}

func (a *Appliance) UpdateAppliance(ctx context.Context, id string, appliance openapi.Appliance) error {
	_, response, err := a.APIClient.AppliancesApi.AppliancesIdPut(ctx, id).Appliance(appliance).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	return nil
}

func (a *Appliance) DisableController(ctx context.Context, id string, appliance openapi.Appliance) error {
	appliance.Controller.Enabled = openapi.PtrBool(false)

	return a.UpdateAppliance(ctx, id, appliance)
}

func (a *Appliance) EnableController(ctx context.Context, id string, appliance openapi.Appliance) error {
	appliance.Controller.Enabled = openapi.PtrBool(true)

	return a.UpdateAppliance(ctx, id, appliance)
}

func (a *Appliance) UpdateMaintenanceMode(ctx context.Context, id string, value bool) (string, error) {
	o := openapi.AppliancesIdMaintenancePostRequest{
		Enabled: value,
	}
	m, response, err := a.APIClient.ApplianceMaintenanceApi.AppliancesIdMaintenancePost(ctx, id).AppliancesIdMaintenancePostRequest(o).Execute()
	if err != nil {
		return "", api.HTTPErrorResponse(response, err)
	}
	return m.GetId(), nil
}

func (a *Appliance) EnableMaintenanceMode(ctx context.Context, id string) (string, error) {
	return a.UpdateMaintenanceMode(ctx, id, true)
}

func (a *Appliance) DisableMaintenanceMode(ctx context.Context, id string) (string, error) {
	return a.UpdateMaintenanceMode(ctx, id, false)
}

func (a *Appliance) UpgradeComplete(ctx context.Context, id string, SwitchPartition bool) error {
	o := openapi.AppliancesIdUpgradeCompletePostRequest{
		SwitchPartition: openapi.PtrBool(SwitchPartition),
	}
	_, response, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradeCompletePost(ctx, id).AppliancesIdUpgradeCompletePostRequest(o).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	return nil
}

func (a *Appliance) UpgradeSwitchPartition(ctx context.Context, id string) error {
	_, response, err := a.APIClient.ApplianceUpgradeApi.AppliancesIdUpgradeSwitchPartitionPost(ctx, id).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	return nil
}

func (a *Appliance) ApplianceSwitchPartition(ctx context.Context, id string) error {
	req := a.APIClient.ApplianceApi.AppliancesIdSwitchPartitionPost(ctx, id)
	_, _, err := req.Execute()
	if err != nil {
		return err
	}
	return nil
}

func (a *Appliance) ForceDisableControllers(ctx context.Context, disable []openapi.Appliance) (*openapi.AppliancesForceDisableControllersPost200Response, string, error) {
	ids := []string{}
	for _, a := range disable {
		ids = append(ids, a.GetId())
	}

	postBody := openapi.AppliancesForceDisableControllersPostRequest{
		ApplianceIds: ids,
	}
	result, response, err := a.APIClient.AppliancesApi.AppliancesForceDisableControllersPost(ctx).AppliancesForceDisableControllersPostRequest(postBody).Execute()
	if err != nil {
		return nil, "", api.HTTPErrorResponse(response, err)
	}
	changeID := response.Header.Get("Change-ID")
	if len(changeID) <= 0 {
		return result, changeID, errors.New("No change ID sent")
	}

	return result, changeID, nil
}

func (a *Appliance) RepartitionIPAllocations(ctx context.Context) (string, error) {
	_, resp, err := a.APIClient.AppliancesApi.AppliancesRepartitionIpAllocationsPost(ctx).Execute()
	if err != nil {
		return "", api.HTTPErrorResponse(resp, err)
	}
	changeID := resp.Header.Get("Change-ID")
	if len(changeID) <= 0 {
		return changeID, errors.New("No change ID sent")
	}

	return changeID, nil
}

func (a *Appliance) ZTPStatus(ctx context.Context) (*openapi.ZtpStatus, error) {
	result, response, err := a.APIClient.ZTPApi.ZtpGet(ctx).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	if result == nil {
		return nil, api.HTTPErrorResponse(response, errors.New("ZtpStatus is nil"))
	}
	return result, nil
}

func (a *Appliance) ZTPUpdateNotify(ctx context.Context) (*openapi.ZtpVersionStatus, error) {
	result, response, err := a.APIClient.ZTPApi.ZtpServicesVersionPost(ctx).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	if result == nil {
		return nil, api.HTTPErrorResponse(response, errors.New("ZtpVersionStatus is nil"))
	}
	return result, nil
}
