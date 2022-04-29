package keyring

import (
	"fmt"

	"github.com/appgate/sdpctl/pkg/hashcode"
	zkeyring "github.com/zalando/go-keyring"
)

const (
	keyringService = "sdpctl"
	password       = "password"
	username       = "username"
	bearer         = "bearer"
)

func format(prefix, value string) string {
	return fmt.Sprintf("%d.%s", hashcode.String(prefix), value)
}

func getSecret(key string) (string, error) {
	return zkeyring.Get(keyringService, key)
}

func setSecret(key, value string) error {
	return zkeyring.Set(keyringService, key, value)
}

func deleteSecret(key string) error {
	return zkeyring.Delete(keyringService, key)
}
