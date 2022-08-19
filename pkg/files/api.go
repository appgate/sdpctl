package files

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
)

type FilesAPI struct {
	Config     *configuration.Config
	HTTPClient *http.Client
}

func (f *FilesAPI) List(ctx context.Context) ([]openapi.File, error) {
	url := fmt.Sprintf("%s/files", f.Config.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	token, err := f.Config.GetBearTokenHeaderValue()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+json", f.Config.Version))

	response, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 400 {
		return nil, errors.New(string(body))
	}

	type responseBody struct {
		Data []openapi.File
	}

	respBody := responseBody{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		return nil, err
	}

	return respBody.Data, nil
}

func (f *FilesAPI) Delete(ctx context.Context, filename string) error {
	url := fmt.Sprintf("%s/files/%s", f.Config.URL, filename)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	token, err := f.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+json", f.Config.Version))

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}

	return nil
}
