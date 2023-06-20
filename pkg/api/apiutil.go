package api

import (
	"encoding/json"
	"errors"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
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

type Error struct {
	StatusCode int
	Err        error
	Errors     []error
}

var (
	ErrFileNotFound       = errors.New("File not found")
	ForbiddenErr    error = &Error{
		StatusCode: 403,
		Err:        errors.New("403 Forbidden"),
	}
	UnavailableErr error = &Error{
		StatusCode: 503,
		Err:        errors.New("503 Service Unavailable"),
	}
)

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("HTTP %d - %s", e.StatusCode, e.Err.Error())
	}
	if len(e.Errors) > 0 {
		return stderrors.Join(e.Errors...).Error()
	}
	return "Internal error"
}

func HTTPErrorResponse(response *http.Response, err error) error {
	if response == nil {
		return fmt.Errorf("No response %w", err)
	}
	ae := &Error{StatusCode: response.StatusCode, Err: err}

	responseBody, errRead := io.ReadAll(response.Body)
	if errRead != nil {
		return ae
	}
	errBody := GenericErrorResponse{}
	if errMarshal := json.Unmarshal(responseBody, &errBody); errMarshal != nil {
		return ae
	}

	if len(errBody.Message) > 0 {
		ae.Errors = append(ae.Errors, stderrors.New(errBody.Message))
	}
	for _, e := range errBody.Errors {
		ae.Errors = append(ae.Errors, fmt.Errorf("%s %s", e.Field, e.Message))

	}
	return ae
}
