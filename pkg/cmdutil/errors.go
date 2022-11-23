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
)
