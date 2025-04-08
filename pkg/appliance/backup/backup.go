package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"
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
	status, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, applianceID).AppliancesIdBackupPostRequest(o).Execute()
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

	client := b.HTTPClient
	cfg := b.APIClient.GetConfig()
	url, err := cfg.ServerURLWithContext(ctx, "ApplianceBackupApiService.AppliancesIdBackupBackupIdGet")
	if err != nil {
		return nil, err
	}
	url = fmt.Sprintf("%s/appliances/%s/backup/%s", url, applianceID, backupID)
	ctx = context.WithValue(ctx, api.ContextAcceptValue, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", b.Version))
	headReq, err := http.NewRequestWithContext(ctx, http.MethodHead, url, http.NoBody)

	if err != nil {
		return nil, err
	}
	head, err := client.Do(headReq)
	if err != nil {
		return nil, err
	}

	size := head.ContentLength
	if err := retryDownload(ctx, client, out, url, size); err != nil {
		return nil, err
	}

	return out, nil
}

func retryDownload(ctx context.Context, client *http.Client, f *os.File, url string, size int64) error {
	start := int64(0)
	return backoff.Retry(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return backoff.Permanent(err)
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
		log := logrus.WithField("request", req)
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		log.WithField("response", res).Debug("download request and response")
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			return fmt.Errorf("response does not indicate success: %v", res.Status)
		}
		f.Seek(start, 0)
		n, err := io.Copy(f, res.Body)
		if err != nil {
			start = start + int64(n)
			return err
		}
		return nil
	}, backoff.NewExponentialBackOff())
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
	status, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdStatusGet(timeoutCTX, applianceID, backupID).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return status, nil
}
