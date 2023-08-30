package cmdutil

import "errors"

var (
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

	ErrDailyVersionCheck = errors.New("version check already done today")
)
