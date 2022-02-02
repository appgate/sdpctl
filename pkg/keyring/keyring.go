package keyring

import (
	"fmt"

	"github.com/appgate/appgatectl/pkg/hashcode"
	zkeyring "github.com/zalando/go-keyring"
)

const (
	keyringService = "appgatectl"
	password       = "password"
	username       = "username"
	bearer         = "bearer"
)

func getSecret(key string) (string, error) {
	secret, err := zkeyring.Get(keyringService, key)
	if err != nil {
		return "", nil
	}
	return secret, nil
}

func format(prefix, value string) string {
	return fmt.Sprintf("%d.%s", hashcode.String(prefix), value)
}

func setSecret(key, value string) error {
	return zkeyring.Set(keyringService, key, value)
}

func GetPassword(prefix string) (string, error) {
	return getSecret(format(prefix, password))
}

func SetPassword(prefix, secret string) error {
	return setSecret(format(prefix, password), secret)
}

func GetBearer(prefix string) (string, error) {
	return getSecret(format(prefix, bearer))
}

func SetBearer(prefix, secret string) error {
	return setSecret(format(prefix, bearer), secret)
}

func SetUsername(prefix, secret string) error {
	return setSecret(format(prefix, username), secret)
}

func GetUsername(prefix string) (string, error) {
	return getSecret(format(prefix, username))
}
