package cmdutil

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/appgate/sdpctl/pkg/api"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

type ExitCode int

var ErrExitAuth = errors.New("no authentication")

const (
	ExitOK     ExitCode = 0
	ExitError  ExitCode = 1
	ExitCancel ExitCode = 2
	ExitAuth   ExitCode = 4
)

func privligeError(cmd *cobra.Command, err *api.Error) error {
	caller := "sdpctl"
	if cmd != nil {
		caller = cmd.Root().Name()
	}
	if err.StatusCode == http.StatusForbidden {
		var result *multierror.Error
		result = multierror.Append(result, fmt.Errorf("Run '%s privileges' to see your current user privileges", caller))
		if err.RequestURL != nil {
			result = multierror.Append(result, errors.New(*err.RequestURL))
		}
		return result
	}
	return nil
}

func ExecuteCommand(cmd *cobra.Command) ExitCode {
	cmd, err := cmd.ExecuteC()
	if err != nil {
		var result *multierror.Error

		if we := errors.Unwrap(err); we != nil {
			// if the command return a api error, (api.HTTPErrorResponse) for example HTTP 400-599, we will
			// resolve each nested error and convert them to multierror to prettify it for the user in a list view.
			if ae, ok := we.(*api.Error); ok {
				result = multierror.Append(result, privligeError(cmd, ae))
				for _, e := range ae.Errors {
					result = multierror.Append(result, e)
				}
			}
			// Unwrap error and check if we have a nested multierr
			// if we do, we will make the errors flat for 1 level
			// otherwise, append error to new multierr list
			if merr, ok := we.(*multierror.Error); ok {
				for _, e := range merr.Errors {
					result = multierror.Append(result, e)
				}
			} else {
				result = multierror.Append(result, err)
			}
		} else {
			if ae, ok := err.(*api.Error); ok {
				result = multierror.Append(result, privligeError(cmd, ae))
				for _, e := range ae.Errors {
					result = multierror.Append(result, e)
				}
			} else {
				result = multierror.Append(result, err)
			}
		}

		// if error is DeadlineExceeded, add custom ErrCommandTimeout
		if errors.Is(err, context.DeadlineExceeded) {
			result = multierror.Append(result, ErrCommandTimeout)
		}

		// if we during any request get a SSL error, (un-trusted certificate) error, prompt the user to import the pem file.
		var sslErr x509.UnknownAuthorityError
		if errors.As(err, &sslErr) {
			result = multierror.Append(result, errors.New("Trust the certificate or import a PEM file using 'sdpctl configure --pem=<path/to/pem>'"))
		}

		// print all multierrors to stderr, then return correct exitcode based on error type
		if result.ErrorOrNil() == nil {
			result = multierror.Append(result, err)
		}
		fmt.Fprintln(cmd.ErrOrStderr(), result.ErrorOrNil())

		if errors.Is(err, ErrExitAuth) {
			return ExitAuth
		}
		if errors.Is(err, ErrExecutionCanceledByUser) {
			return ExitCancel
		}
		// only show usage prompt if we get invalid args / flags
		errorString := err.Error()
		if strings.Contains(errorString, "arg(s)") || strings.Contains(errorString, "flag") || strings.Contains(errorString, "command") {
			fmt.Fprintln(cmd.ErrOrStderr())
			fmt.Fprintln(cmd.ErrOrStderr(), cmd.UsageString())
			return ExitError
		}

		return ExitError
	}
	return ExitOK
}
