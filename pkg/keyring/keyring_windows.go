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
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/billgraziano/dpapi"
)

func ClearCredentials(prefix string) error {
	for _, k := range []string{username, password} {
		if err := deleteSecret(format(prefix, k)); err != nil {
			return err
		}
	}
	p, err := filepath.Abs(fmt.Sprintf("%s/%s", filesystem.ConfigDir(), format(prefix, bearer)))
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
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

	p, err := filepath.Abs(fmt.Sprintf("%s/%s", filesystem.ConfigDir(), format(prefix, bearer)))
	if err != nil {
		return "", err
	}

	dat, err := os.ReadFile(p)
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
	encrypted, err := dpapi.EncryptBytes([]byte(secret))
	if err != nil {
		return fmt.Errorf("could not encrypt bearer token to Windows DPAPI %w", err)
	}
	dir := filesystem.ConfigDir()
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	p, err := filepath.Abs(fmt.Sprintf("%s/%s", dir, format(prefix, bearer)))
	if err != nil {
		return err
	}

	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("could create file %w", err)
	}
	defer f.Close()
	if _, err := f.Write(encrypted); err != nil {
		return fmt.Errorf("could write file %w", err)
	}
	return nil
}
