//go:build windows
// +build windows

// The bearer token length is too long to store in Windows Credential Manager API
// so for Windows, we will store and fetch the bearer token from file.
// the content of the file will be encrypted with The Windows DPAPI
//
// By default, the bearer token file will be located in %APPDATA%/sdpctl
// if not overwritten by SDPCTL_CONFIG_DIR
//
package keyring

import (
	"fmt"
	"os"

	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/billgraziano/dpapi"
)

func GetPassword(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_PASSWORD"); ok {
		return v, nil
	}
	return getSecret(format(prefix, password))
}

func SetPassword(prefix, secret string) error {
	return setSecret(format(prefix, password), secret)
}

func SetUsername(prefix, secret string) error {
	return setSecret(format(prefix, username), secret)
}

func GetUsername(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_USERNAME"); ok {
		return v, nil
	}
	return getSecret(format(prefix, username))
}

func GetBearer(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_BEARER"); ok {
		return v, nil
	}
	filepath := filesystem.ConfigDir()
	filename := format(prefix, bearer)

	dat, err := os.ReadFile(fmt.Sprintf("%s/%s", filepath, filename))
	if err != nil {
		return "", err
	}
	dec, err := dpapi.DecryptBytes(dat)
	if err != nil {
		return "", fmt.Errorf("could not decrypt bearer token from Windows DPAPI %w", err)
	}
	return string(dec), nil
}

func SetBearer(prefix, secret string) error {
	filepath := filesystem.ConfigDir()
	filename := format(prefix, bearer)

	encrypted, err := dpapi.EncryptBytes([]byte(secret))
	if err != nil {
		return fmt.Errorf("could not encrypt bearer token to Windows DPAPI %w", err)
	}
	f, err := os.Create(fmt.Sprintf("%s/%s", filepath, filename))
	if err != nil {
		return fmt.Errorf("could create file %w", err)
	}
	defer f.Close()
	if _, err := f.Write(encrypted); err != nil {
		return fmt.Errorf("could write file %w", err)
	}
	return nil
}
