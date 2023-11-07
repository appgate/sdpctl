package api

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/cenkalti/backoff/v4"
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
	RequestURL *string
	Errors     []error
}

type ContextKey string

const ContextAcceptValue ContextKey = "Accept"

var (
	ErrFileNotFound       = stderrors.New("File not found")
	ForbiddenErr    error = &Error{
		StatusCode: 403,
		Err:        stderrors.New("403 Forbidden"),
	}
	UnavailableErr error = &Error{
		StatusCode: 503,
		Err:        stderrors.New("503 Service Unavailable"),
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

	if response.Request != nil {
		var ptr = new(string)
		*ptr = fmt.Sprintf("HTTP %s %s", response.Request.Method, response.Request.URL)
		ae.RequestURL = ptr
	}

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

func RequestRetry(c *http.Client, req *http.Request) (*http.Response, error) {
	return backoff.RetryWithData(func() (*http.Response, error) {
		res, err := c.Do(req)
		if err != nil {
			return nil, backoff.Permanent(err)
		}
		if res.StatusCode >= 300 {
			if res.StatusCode == http.StatusNotFound {
				return nil, backoff.Permanent(fmt.Errorf("%s not found", req.URL))
			}
			return nil, fmt.Errorf("recieved %s status", res.Status)
		}
		return res, nil
	}, backoff.NewExponentialBackOff())
}
