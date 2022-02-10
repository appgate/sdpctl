package api

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
)

type GenericErrorResponse struct {
	ID      string   `json:"id,omitempty"`
	Message string   `json:"message,omitempty"`
	Errors  []Errors `json:"errors,omitempty"`
}
type Errors struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func HTTPErrorResponse(response *http.Response, err error) error {
	if response == nil {
		return fmt.Errorf("No response %w", err)
	}
	var errors error
	if util.InBetween(response.StatusCode, 400, 499) {
		responseBody, errRead := io.ReadAll(response.Body)
		if errRead != nil {
			return fmt.Errorf("%d Could not read response body %w", response.StatusCode, err)
		}
		errBody := GenericErrorResponse{}
		if err := json.Unmarshal(responseBody, &errBody); err != nil {
			return fmt.Errorf("%d %w", response.StatusCode, err)
		}
		errors = multierror.Append(errors, stderrors.New(errBody.Message))
		for _, e := range errBody.Errors {
			errors = multierror.Append(errors, fmt.Errorf("%s %s", e.Field, e.Message))
		}
		return errors
	}
	return multierror.Append(errors, err)
}
