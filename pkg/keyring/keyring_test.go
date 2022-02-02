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
