package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
)

type Backup struct {
	configuration *configuration.Config
	HTTPClient    *http.Client
	APIClient     *openapi.APIClient
	Token         string
	Version       int
}

func New(h *http.Client, c *openapi.APIClient, config *configuration.Config, token string) *Backup {
	return &Backup{
		APIClient:     c,
		HTTPClient:    h,
		configuration: config,
		Token:         token,
		Version:       config.Version,
	}
}

// Initiate an appliance backup. The progress can be followed by polling the appliance via "GET appliances/{id}/backup/{backupId}/status".
func (b *Backup) Initiate(ctx context.Context, applianceID string, logs, audit bool) (string, error) {
	o := openapi.AppliancesIdBackupPostRequest{
		Logs:  &logs,
		Audit: &audit,
	}
	status, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, applianceID).Authorization(b.Token).AppliancesIdBackupPostRequest(o).Execute()
	if err != nil {
		if response.StatusCode == http.StatusServiceUnavailable {
			return "", api.UnavailableErr
		}
		return "", api.HTTPErrorResponse(response, err)
	}

	return status.GetId(), nil
}

// DownloadLegacy a completed appliance backup with the given ID of an Appliance
func (b *Backup) DownloadLegacy(ctx context.Context, applianceID, backupID, destination string) (*os.File, error) {
	client := b.HTTPClient
	cfg := b.APIClient.GetConfig()
	url, err := cfg.ServerURLWithContext(ctx, "ApplianceBackupApiService.AppliancesIdBackupBackupIdGet")
	if err != nil {
		return nil, err
	}
	url = fmt.Sprintf("%s/appliances/%s/backup/%s", url, applianceID, backupID)
	ctx = context.WithValue(ctx, api.ContextAcceptValue, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", b.Version))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Close = true

	out, err := os.Create(destination)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	res, err := client.Do(req)
	if err != nil {
		return nil, api.HTTPErrorResponse(res, err)
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return nil, api.HTTPErrorResponse(res, errors.New("unexpected response code"))
	}

	_, err = out.ReadFrom(res.Body)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Download a completed appliance backup with the given ID of an Appliance
func (b *Backup) Download(ctx context.Context, applianceID, backupID, destination string) (*os.File, error) {
	out, err := os.Create(destination)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	w := io.NewOffsetWriter(out, 0)

	client := b.HTTPClient
	cfg := b.APIClient.GetConfig()
	url, err := cfg.ServerURLWithContext(ctx, "ApplianceBackupApiService.AppliancesIdBackupBackupIdGet")
	if err != nil {
		return nil, err
	}
	url = fmt.Sprintf("%s/appliances/%s/backup/%s", url, applianceID, backupID)
	ctx = context.WithValue(ctx, api.ContextAcceptValue, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", b.Version))
	written := 0
	maxRetries := 10
	retryCount := 0

RETRY:
	offset := int64(written)
	if written > 0 {
		offset++
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		if retryCount <= maxRetries {
			retryCount++
			goto RETRY
		}
		return nil, err
	}
	req.Close = true
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-", offset))

	res, err := client.Do(req)
	if err != nil {
		if retryCount <= maxRetries {
			retryCount++
			time.Sleep(time.Second)
			goto RETRY
		}
		return nil, api.HTTPErrorResponse(res, err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		if retryCount <= maxRetries {
			retryCount++
			time.Sleep(time.Second)
			goto RETRY
		}
		return nil, api.HTTPErrorResponse(res, errors.New("unexpected response code"))
	}
	cr := res.Header.Get("Content-Range")
	totalSize, _ := strconv.ParseInt(strings.Split(cr, "/")[1], 10, 64)

	body := []byte{}
	n, err := io.ReadAtLeast(res.Body, body, int(totalSize))
	written = written + n
	if err != nil {
		if retryCount <= maxRetries {
			retryCount++
			time.Sleep(time.Second)
			goto RETRY
		}
		return nil, err
	}
	n, err = w.WriteAt(body, offset)
	written = written + n
	if err != nil {
		if retryCount <= maxRetries {
			retryCount++
			time.Sleep(time.Second)
			goto RETRY
		}
		return nil, err
	}
	if int64(written) != totalSize {
		return nil, fmt.Errorf("incomplete download - total size: %d, downloaded: %d", totalSize, written)
	}
	return out, nil
}

const (
	// https://github.com/appgate/sdp-api-specification/blob/0cae2de511a135ca1c29beb89fe9d38e83ffc4f1/appliance_backup.yml#L87-L88
	Processing string = "processing"
	Done       string = "done"
	Success    string = "success"
	Failure    string = "failure"
)

func (b *Backup) Status(ctx context.Context, applianceID, backupID string) (*openapi.AppliancesIdBackupBackupIdStatusGet200Response, error) {
	timeoutCTX, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	status, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdStatusGet(timeoutCTX, applianceID, backupID).Authorization(b.Token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return status, nil
}
