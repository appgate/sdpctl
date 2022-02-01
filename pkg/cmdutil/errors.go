package cmdutil

import "errors"

var (
	// ErrExecutedOnAppliance signals when the program is executed within a appliance
	ErrExecutedOnAppliance = errors.New("This should not be executed on an appliance")
	// ErrExecutionCanceledByUser signals user-initiated cancellation
	ErrExecutionCanceledByUser = errors.New("Cancelled by user")
)
