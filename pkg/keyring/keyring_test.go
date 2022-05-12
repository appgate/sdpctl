package keyring

import (
	"testing"

	zkeyring "github.com/zalando/go-keyring"
)

func TestSetSecretAndGetSecret(t *testing.T) {
	zkeyring.MockInit()
	if err := setSecret("foo", "bar"); err != nil {
		t.Errorf("setSecret() Got error = %v, wantErr none", err)
	}
	secret, err := getSecret("foo")
	if err != nil {
		t.Errorf("GetSecret() got error %v, want none", err)
	}
	if secret != "bar" {
		t.Fatalf("got secret, wrong value, expected bar, got %s", secret)
	}
}

func TestDeleteSecret(t *testing.T) {
	zkeyring.MockInit()
	if err := setSecret("foo", "bar"); err != nil {
		t.Errorf("setSecret() Got error = %v, wantErr none", err)
	}
	_, err := getSecret("foo")
	if err != nil {
		t.Errorf("GetSecret() got error %v, want none", err)
	}
	deleteSecret("foo")
	if _, err = getSecret("foo"); err == nil {
		t.Errorf("deleteSecret() got no error, want error")
	}
}

func TestClearCredentials(t *testing.T) {
	zkeyring.MockInit()
	var (
		prefix   = "test-unit"
		username = "user"
		password = "password"
		bearer   = "somebearer"
	)
	if err := SetUsername(prefix, username); err != nil {
		t.Error("TEST FAIL: failed to set username", err)
	}
	if err := SetPassword(prefix, password); err != nil {
		t.Error("TEST FAIL: failed to set password", err)
	}
	if err := SetBearer(prefix, bearer); err != nil {
		t.Error("TEST FAIL: failed to set bearer", err)
	}

	if err := ClearCredentials(prefix); err != nil {
		t.Fatalf("failed to clear credentials %s", err)
	}

	if _, err := GetUsername(prefix); err == nil {
		t.Error("TEST FAIL: failed to remove username", err)
	}
	if _, err := GetPassword(prefix); err == nil {
		t.Error("TEST FAIL: failed to remove password", err)
	}
	if _, err := GetBearer(prefix); err == nil {
		t.Error("TEST FAIL: failed to remove bearer", err)
	}
}
