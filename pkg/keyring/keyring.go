//go:build !windows && !darwin
// +build !windows,!darwin

package keyring

import (
	"errors"
	"fmt"
	"os"
	"strings"

	zkeyring "github.com/zalando/go-keyring"
)

const (
	// Error string if org.freedesktop.secrets does not exists, for example a environment
	// without X (graphical interface, for example a server environment)
	secretMissing = "org.freedesktop.secrets was not provided by any"
)

// ClearCredentials removes any existing items in the keychain,
// it will ignore if not found errors
func ClearCredentials(prefix string) error {
	for _, k := range []string{username, password} {
		if err := deleteSecret(format(prefix, k)); err != nil {
			if !errors.Is(err, zkeyring.ErrNotFound) {
				return err
			}
		}
	}
	if err := DeleteBearer(prefix); err != nil {
		return err
	}
	return nil
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
		return os.Setenv("SDPCTL_PASSWORD", secret)
	}
	return err
}

func GetBearer(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_BEARER"); ok {
		return v, nil
	}
	v, err := getSecret(format(prefix, bearer))
	if err != nil {
		return "", fmt.Errorf("could not retrieve bearer token for %s configuration, run 'sdpctl configure login' or set SDPCTL_BEARER %w", prefix, err)
	}
	return v, nil
}

func SetBearer(prefix, secret string) error {
	if err := setSecret(format(prefix, bearer), secret); err != nil {
		return os.Setenv("SDPCTL_BEARER", secret)
	}
	return nil
}

func DeleteBearer(prefix string) error {
	if err := deleteSecret(format(prefix, bearer)); err != nil {
		if err != zkeyring.ErrNotFound {
			return err
		}
	}
	if _, ok := os.LookupEnv("SDPCTL_BEARER"); ok {
		return os.Unsetenv("SDPCTL_BEARER")
	}
	return nil
}

func GetRefreshToken(prefix string) (string, error) {
	return getSecret(format(prefix, refreshToken))
}

func SetRefreshToken(prefix, secret string) error {
	return setSecret(format(prefix, refreshToken), secret)
}

func SetUsername(prefix, secret string) error {
	err := setSecret(format(prefix, username), secret)
	if err != nil && strings.Contains(err.Error(), secretMissing) {
		return os.Setenv("SDPCTL_USERNAME", secret)
	}
	return err
}

func GetUsername(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_USERNAME"); ok {
		return v, nil
	}
	return getSecret(format(prefix, username))
}
