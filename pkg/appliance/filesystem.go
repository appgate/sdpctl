package appliance

import (
	"errors"
	"os"
)

var (
	ErrExecutedOnAppliance     = errors.New("This should not be executed on an appliance")
	ErrExecutionCanceledByUser = errors.New("Cancelled by user")
)

func IsOnAppliance() bool {
	if _, err := os.Stat("/mnt/state/config"); os.IsNotExist(err) {
		return false
	}
	return true
}

func FileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
