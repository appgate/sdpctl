package cmdutil

import (
	"errors"
	"fmt"
)

var (
	// GenericErrorWrap wraps an error along with a message
	// The message is intended to be specific to a certain operation the user is trying to perform while the error is more generic
	GenericErrorWrap = func(msg string, err error) error {
		return fmt.Errorf("%s: %w", msg, err)
	}
	// ErrUnexpectedResponseStatus is used for generic response statuses that are unexpected, but not considered errors
	ErrUnexpectedResponseStatus = func(want, got int) error {
		return fmt.Errorf("unexpected response status - want %d, got %d", want, got)
	}
	// ErrExecutedOnAppliance signals when the program is executed within a appliance
	ErrExecutedOnAppliance = errors.New("This should not be executed on an appliance")
	// ErrExecutionCanceledByUser signals user-initiated cancellation
	ErrExecutionCanceledByUser = errors.New("Cancelled by user")
	// ErrCommandTimeout is used instead of the default 'Context exceeded deadline' when command times out
	ErrCommandTimeout = errors.New("Command timed out")
	// ErrMissingTTY is used when no TTY is available
	ErrMissingTTY = errors.New("No TTY present")
	// ErrNetworkError is used when a command encounters multiple timeouts on a request, which we will presume is a network error
	ErrNetworkError = errors.New("Failed to communicate with SDP Collective. Bad connection")
	// ErrSSL is used when a certificate is not provided or is invalid
	ErrSSL = errors.New("Trust the certificate or import a PEM file using 'sdpctl configure --pem=<path/to/pem>'")
	// ErrNothingToPrepare is used when there are no appliances to prepare for upgrade
	ErrNothingToPrepare = errors.New("No appliances to prepare for upgrade. All appliances may have been filtered or are already prepared. See the log for more details")
	// ErrDailyVersionCheck is used when version check has already been done recently
	ErrDailyVersionCheck = errors.New("version check already done today")
	// ErrVersionCheckDisabled is used when version check has been disabled
	ErrVersionCheckDisabled = errors.New("version check disabled")
	// ErrUnsupportedOperation is used when the user tries to do an operation that is unsupported by the Appliance, likely due to version
	ErrUnsupportedOperation = errors.New("Operation not supported on your appliance version")
	// ErrControllerMaintenanceMode is used when trying to connect to appliance where maintenance mode is enabled
	ErrControllerMaintenanceMode = errors.New("controller seem to be in maintenance mode")
	// ErrNoUpgradeAvailable is used when there are no upgrades available for the appliance
	ErrNoUpgradeAvailable = errors.New("Could not perform upgrade.\nAppliances are already running a version higher or equal to the version provided")
)
