//go:build darwin
// +build darwin

// The bearer token is not saved in the OS keychain for macOS because of string length limitation.
// If the user choose to save their username/password in the keychain, they will still experiance a
// seemless integration, however they need todo a few more http requests on startup for each command they execute.
package keyring

import (
	"errors"
	"os"

	zkeyring "github.com/zalando/go-keyring"
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
	if err != nil {
		os.Setenv("SDPCTL_PASSWORD", secret)
		return nil
	}
	return err
}

func GetBearer(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_BEARER"); ok {
		return v, nil
	}
	return "", errors.New("bearer token not saved persistently on macOS")
}

func SetBearer(prefix, secret string) error {
	os.Setenv("SDPCTL_BEARER", secret)
	return nil
}

func SetUsername(prefix, secret string) error {
	err := setSecret(format(prefix, username), secret)
	if err != nil {
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

func GetRefreshToken(prefix string) (string, error) {
	return "", errors.New("macOS is not supported")
}

func SetRefreshToken(prefix, secret string) error {
	return errors.New("macOS is not supported")
}
