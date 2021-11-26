package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/appgate/appgatectl/pkg/util"
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
	if util.InBetween(response.StatusCode, 400, 499) {
		responseBody, errRead := io.ReadAll(response.Body)
		if errRead != nil {
			return errRead
		}
		errBody := GenericErrorResponse{}
		if err := json.Unmarshal(responseBody, &errBody); err != nil {
			return err
		}
		// TODO custom error on http.StatusConflict,  StatusNotAcceptable ?
		s := errBody.Message
		for _, e := range errBody.Errors {
			s = s + " " + e.Field + " " + e.Message
		}
		return errors.New(s)
	}
	return nil
}
