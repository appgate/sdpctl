//go:build darwin
// +build darwin

package keyring

import (
	"errors"
	"fmt"
	"github.com/keybase/go-keychain"
	"os"
)

func deleteSecretKey(prefix, name string) error {
	key := format(prefix, name)
	if _, err := QueryKeychain(key); err == nil {
		item := keychain.NewItem()
		item.SetSecClass(keychain.SecClassGenericPassword)
		item.SetService(keyringService)
		item.SetAccount(key)
		err := keychain.DeleteItem(item)
		if err != nil {
			if !errors.Is(err, keychain.ErrorItemNotFound) {
				return errors.New("failed to delete credential from keychain")
			}
		}
	}
	return nil
}

// ClearCredentials removes any existing items in the keychain,
// it will ignore if not found errors
func ClearCredentials(prefix string) error {
	for _, k := range []string{username, password} {
		if err := deleteSecretKey(prefix, k); err != nil {
			return err
		}
	}
	if err := DeleteBearer(prefix); err != nil {
		return err
	}
	return nil
}

func QueryKeychain(key string) (string, error) {
	query := keychain.NewItem()
	query.SetService(keyringService)
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetAccessGroup(keyringService)
	query.SetAccount(key)
	query.SetReturnAttributes(true)
	query.SetReturnData(true)
	result, err := keychain.QueryItem(query)
	if err != nil {
		return "", errors.New("encountered error when querying the keychain")
	}
	if len(result) != 1 {
		return "", errors.New(fmt.Sprintf("could not find key: %s", key))
	}
	return string(result[0].Data), nil
}

func AddKeychain(key string, value string) error {
	item := keychain.NewItem()
	item.SetService(keyringService)
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetMatchLimit(keychain.MatchLimitOne)
	item.SetAccessGroup(keyringService)
	item.SetAccount(key)
	item.SetData([]byte(value))
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	err := keychain.AddItem(item)

	// Overwrite the item if it already exists
	if err == keychain.ErrorDuplicateItem {
		return UpdateKeychain(item, key)
	}

	if err != nil {
		return err
	}
	return nil
}

func UpdateKeychain(updateItem keychain.Item, key string) error {
	queryItem := keychain.NewItem()
	queryItem.SetService(keyringService)
	queryItem.SetSecClass(keychain.SecClassGenericPassword)
	queryItem.SetMatchLimit(keychain.MatchLimitOne)
	queryItem.SetAccessGroup(keyringService)
	queryItem.SetAccount(key)
	queryItem.SetReturnAttributes(true)

	results, err := keychain.QueryItem(queryItem)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to query keychain: %s", err))
	}
	if len(results) != 1 {
		return errors.New(fmt.Sprintf("could not find key: %s", key))
	}

	if err = keychain.UpdateItem(queryItem, updateItem); err != nil {
		return errors.New(fmt.Sprintf("failed to update item: %s", err))
	}

	return nil
}

func GetPassword(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_PASSWORD"); ok {
		return v, nil
	}
	pw, err := QueryKeychain(format(prefix, password))
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed to get password from keychain: %s", err))
	}
	return pw, nil
}

func SetPassword(prefix, secret string) error {
	err := AddKeychain(format(prefix, password), secret)
	if err != nil {
		return err
	}
	return nil
}

func GetBearer(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_BEARER"); ok {
		return v, nil
	}
	token, err := QueryKeychain(format(prefix, bearer))
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed to get bearer token from keychain: %s", err))
	}
	return token, nil
}

func SetBearer(prefix, secret string) error {
	err := AddKeychain(format(prefix, bearer), secret)
	if err != nil {
		return err
	}
	return nil
}

func DeleteBearer(prefix string) error {
	if err := deleteSecretKey(prefix, bearer); err != nil {
		return err
	}
	if _, ok := os.LookupEnv("SDPCTL_BEARER"); ok {
		os.Unsetenv("SDPCTL_BEARER")
	}
	return nil
}

func SetUsername(prefix, secret string) error {
	err := AddKeychain(format(prefix, username), secret)
	if err != nil {
		return err
	}
	return nil
}

func GetUsername(prefix string) (string, error) {
	if v, ok := os.LookupEnv("SDPCTL_USERNAME"); ok {
		return v, nil
	}
	user, err := QueryKeychain(format(prefix, username))
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed to get username from keychain: %s", err))
	}
	return user, nil
}

func GetRefreshToken(prefix string) (string, error) {
	token, err := QueryKeychain(format(prefix, refreshToken))
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed to get refresh token from keychain: %s", err))
	}
	return token, nil
}

func SetRefreshToken(prefix, secret string) error {
	err := AddKeychain(format(prefix, refreshToken), secret)
	if err != nil {
		return err
	}
	return nil
}
