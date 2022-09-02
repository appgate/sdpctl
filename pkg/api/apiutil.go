package api

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"

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

	responseBody, errRead := io.ReadAll(response.Body)
	if errRead != nil {
		return fmt.Errorf("%d Could not read response body %w", response.StatusCode, err)
	}
	errBody := GenericErrorResponse{}
	if errMarshal := json.Unmarshal(responseBody, &errBody); errMarshal != nil {
		return fmt.Errorf("HTTP %d - %w", response.StatusCode, err)
	}
	if len(errBody.Message) > 0 {
		errors = multierror.Append(errors, stderrors.New(errBody.Message))
	}
	for _, e := range errBody.Errors {
		errors = multierror.Append(errors, fmt.Errorf("%s %s", e.Field, e.Message))
	}
	if errors == nil {
		errors = multierror.Append(errors, err)
	}
	return errors
}
