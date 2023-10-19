package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
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
	status, response, err := b.APIClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, applianceID).Authorization(b.Token).AppliancesIdBackupPostRequest(o).Execute()
	if err != nil {
		if response.StatusCode == http.StatusServiceUnavailable {
			return "", api.UnavailableErr
		}
		return "", api.HTTPErrorResponse(response, err)
	}

	return status.GetId(), nil
}

// Download a completed appliance backup with the given ID of an Appliance
func (b *Backup) Download(ctx context.Context, applianceID, backupID, destination string) (*os.File, error) {
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

func (b *Backup) ChunkedDownload(ctx context.Context, applianceID, backupID, destination string) (*os.File, error) {
	type part struct {
		Index int
		Start int64
		Data  []byte
		Error error
	}
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

	getChunk := func(wg *sync.WaitGroup, index int, start, end int64, c chan part) {
		defer wg.Done()
		dataRange := fmt.Sprintf("bytes=%d-%d", start, end)
		if end == 0 {
			dataRange = fmt.Sprintf("bytes=%d-", start)
		}
		logEntry := logrus.WithFields(logrus.Fields{
			"index": index,
			"start": start,
			"end":   end,
		})
		logEntry.Debug("downloading range")

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			c <- part{Error: err}
			return
		}
		req.Header.Add("Range", dataRange)
	REQUEST_CHUNK:
		res, err := client.Do(req)
		if err != nil {
			c <- part{Error: err}
			return
		}
		defer res.Body.Close()
		if res.StatusCode >= 300 {
			err = errors.New("unexpected response")
			c <- part{Error: api.HTTPErrorResponse(res, err)}
			return
		}
		data, err := io.ReadAll(res.Body)
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				goto REQUEST_CHUNK
			}
			c <- part{Error: err}
			return
		}
		c <- part{
			Start: start,
			Index: index,
			Data:  data,
		}
	}

	var chunks int64
	var chunkSize int64 = 1024 << 4

	if header, ok := head.Header["Content-Length"]; ok {
		filesize, err := strconv.Atoi(header[0])
		if err != nil {
			return nil, err
		}

		chunks = int64(filesize) / chunkSize
		if int64(filesize)%chunkSize != 0 {
			chunks++
		}
	}

	dir := filepath.Base(destination)
	ok, err := util.FileExists(dir)
	if err != nil {
		return nil, err
	}
	if !ok {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
	}
	file, err := os.Create(destination)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	parts := make(chan part)
	var wg sync.WaitGroup
	wg.Add(int(chunks))
	for i := 1; i <= int(chunks); i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == int(chunks) {
			end = 0
		}
		go getChunk(&wg, i, start, end, parts)
	}

	go func(wg *sync.WaitGroup, parts chan part) {
		wg.Wait()
		close(parts)
	}(&wg, parts)

	var errs *multierror.Error
	for part := range parts {
		if part.Error != nil {
			logrus.WithError(part.Error).Error("chunk download failed")
			errs = multierror.Append(errs, part.Error)
			continue
		}
		if _, err := file.WriteAt(part.Data, part.Start); err != nil {
			logrus.WithError(err).Error("failed to write chunk")
			errs = multierror.Append(errs, err)
		}
	}

	if errs.Len() > 0 {
		os.Remove(file.Name())
	}

	return file, errs.ErrorOrNil()
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
