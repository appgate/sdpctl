package keyring

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/appgate/sdpctl/pkg/hashcode"
	zkeyring "github.com/zalando/go-keyring"
)

const (
	keyringService = "sdpctl"
	password       = "password"
	username       = "username"
	bearer         = "bearer"
	refreshToken   = "refreshToken"
)

// ErrKeyringTimeOut is returned when the keyring operation takes too long to complete.
var ErrKeyringTimeOut = errors.New("keyring: timeout")

// keyringTimeout max time for a keyring syscall
var keyringTimeout = time.Second * 5

func runWithTimeout(task func() error) error {
	if len(os.Getenv("SDPCTL_NO_KEYRING")) > 0 {
		return nil
	}

	ch := make(chan error)
	go func() {
		ch <- task()
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(keyringTimeout):
		return ErrKeyringTimeOut
	}
}

func format(prefix, value string) string {
	return fmt.Sprintf("%d.%s", hashcode.String(prefix), value)
}

func getSecret(key string) (string, error) {
	if len(os.Getenv("SDPCTL_NO_KEYRING")) > 0 {
		return "", nil
	}
	ch := make(chan struct {
		response string
		err      error
	})
	go func() {
		v, err := zkeyring.Get(keyringService, key)
		ch <- struct {
			response string
			err      error
		}{v, err}
	}()

	select {
	case result := <-ch:
		return result.response, result.err
	case <-time.After(keyringTimeout):
		return "", ErrKeyringTimeOut
	}
}

func setSecret(key, value string) error {
	return runWithTimeout(func() error {
		return zkeyring.Set(keyringService, key, value)
	})
}

func deleteSecret(key string) error {
	return runWithTimeout(func() error {
		return zkeyring.Delete(keyringService, key)
	})
}
