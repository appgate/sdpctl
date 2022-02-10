//go:build !windows
// +build !windows

package keyring

import (
	"os"
	"strings"
)

const (
	// Error string if org.freedesktop.secrets does not exists, for example a environment
	// without X (grapgical interface, for example a server environment)
	secretMissing = "org.freedesktop.secrets was not provided by any"
)

func GetPassword(prefix string) (string, error) {
	if v, ok := os.LookupEnv("APPGATECTL_PASSWORD"); ok {
		return v, nil
	}
	return getSecret(format(prefix, password))
}

func SetPassword(prefix, secret string) error {
	err := setSecret(format(prefix, password), secret)
	if err != nil && strings.Contains(err.Error(), secretMissing) {
		os.Setenv("APPGATECTL_PASSWORD", secret)
		return nil
	}
	return err
}

func GetBearer(prefix string) (string, error) {
	if v, ok := os.LookupEnv("APPGATECTL_BEARER"); ok {
		return v, nil
	}
	return getSecret(format(prefix, bearer))
}

func SetBearer(prefix, secret string) error {
	err := setSecret(format(prefix, bearer), secret)
	if err != nil && strings.Contains(err.Error(), secretMissing) {
		os.Setenv("APPGATECTL_BEARER", secret)
		return nil
	}
	return err
}

func SetUsername(prefix, secret string) error {
	err := setSecret(format(prefix, username), secret)
	if err != nil && strings.Contains(err.Error(), secretMissing) {
		os.Setenv("APPGATECTL_USERNAME", secret)
		return nil
	}
	return err
}

func GetUsername(prefix string) (string, error) {
	if v, ok := os.LookupEnv("APPGATECTL_USERNAME"); ok {
		return v, nil
	}
	return getSecret(format(prefix, username))
}
