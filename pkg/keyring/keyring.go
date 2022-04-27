//go:build !windows
// +build !windows

package keyring

import (
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
)

const (
	// Error string if org.freedesktop.secrets does not exists, for example a environment
	// without X (grapgical interface, for example a server environment)
	secretMissing = "org.freedesktop.secrets was not provided by any"
)

func ClearCredentials(prefix string) error {
	var errs error
	for _, k := range []string{username, password, bearer} {
		if err := deleteSecret(format(prefix, k)); err != nil {
			errs = multierror.Append(err)
		}
	}
	return errs
}

func GetPassword(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_PASSWORD"); ok {
		return v, nil
	}
	return getSecret(format(prefix, password))
}

func SetPassword(prefix, secret string) error {
	err := setSecret(format(prefix, password), secret)
	if err != nil && strings.Contains(err.Error(), secretMissing) {
		os.Setenv("SDPCTL_PASSWORD", secret)
		return nil
	}
	return err
}

func GetBearer(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_BEARER"); ok {
		return v, nil
	}
	return getSecret(format(prefix, bearer))
}

func SetBearer(prefix, secret string) error {
	err := setSecret(format(prefix, bearer), secret)
	if err != nil && strings.Contains(err.Error(), secretMissing) {
		os.Setenv("SDPCTL_BEARER", secret)
		return nil
	}
	return err
}

func SetUsername(prefix, secret string) error {
	err := setSecret(format(prefix, username), secret)
	if err != nil && strings.Contains(err.Error(), secretMissing) {
		os.Setenv("SDPCTL_USERNAME", secret)
		return nil
	}
	return err
}

func GetUsername(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_USERNAME"); ok {
		return v, nil
	}
	return getSecret(format(prefix, username))
}
