package keyring

import zkeyring "github.com/zalando/go-keyring"

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

func setSecret(key, value string) error {
	return zkeyring.Set(keyringService, key, value)
}

func GetPassword() (string, error) {
	return getSecret(password)
}

func SetPassword(secret string) error {
	return setSecret(password, secret)
}

func GetBearer() (string, error) {
	return getSecret(bearer)
}

func SetBearer(secret string) error {
	return setSecret(bearer, secret)
}

func SetUsername(secret string) error {
	return setSecret(username, secret)
}

func GetUsername() (string, error) {
	return getSecret(username)
}
