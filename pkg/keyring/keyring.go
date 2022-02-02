//go:build !windows
// +build !windows

package keyring

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
