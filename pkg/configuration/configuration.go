package configuration

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/appgate/appgatectl/pkg/keyring"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type Config struct {
	URL                      string `mapstructure:"url"`
	Provider                 string `mapstructure:"provider"`
	Insecure                 bool   `mapstructure:"insecure"`
	Debug                    bool   `mapstructure:"debug"`       // http debug flag
	Version                  int    `mapstructure:"api_version"` // api peer interface version
	BearerToken              string `mapstructure:"bearer"`      // current logged in user token
	ExpiresAt                string `mapstructure:"expires_at"`
	DeviceID                 string `mapstructure:"device_id"`
	PemFilePath              string `mapstructure:"pem_filepath"`
	PrimaryControllerVersion string `mapstructure:"primary_controller_version"`
	Timeout                  int    // HTTP timeout, not supported in the config file.
}

type Credentials struct {
	Username string
	Password string
}

func (c *Config) GetBearTokenHeaderValue() string {
	h, _ := c.GetHost() // TODO catch err, update function signature (string, error)
	if v, err := keyring.GetBearer(h); err == nil {
		return fmt.Sprintf("Bearer %s", v)
	}
	return ""
}

// DefaultDeviceID return a unique ID in UUID format.
// machine.ID() tries to read
// /etc/machine-id on Linux
// /etc/hostid on BSD
// ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID on OSX
// reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid on Windows
// and tries to parse the value as a UUID
// https://github.com/denisbrodbeck/machineid
// if we can't get a valid UUID based on the machine ID, we will fallback to a random UUID value.
func DefaultDeviceID() string {
	readAndParseUUID := func() (string, error) {
		id, err := machineid.ID()
		if err != nil {
			return "", err
		}
		uid, err := uuid.Parse(id)
		if err != nil {
			return "", err
		}
		return uid.String(), nil
	}
	v, err := readAndParseUUID()
	if err != nil {
		return uuid.New().String()
	}
	return v
}

func IsAuthCheckEnabled(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return false
	}
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations["skipAuthCheck"] == "true" {
			return false
		}
	}
	return true
}

var ErrNoAddr = errors.New("no valid address set, run 'appgatectl configure' or set APPGATECTL_URL")

func NormalizeURL(u string) (string, error) {
	if len(u) <= 0 {
		return "", ErrNoAddr
	}
	if r := regexp.MustCompile(`^https?://`); !r.MatchString(u) {
		u = fmt.Sprintf("https://%s", u)
	}
	url, err := url.ParseRequestURI(u)
	if err != nil {
		return "", err
	}
	if url.Scheme != "https" {
		url.Scheme = "https"
	}
	if len(url.Port()) <= 0 {
		url.Host = fmt.Sprintf("%s:%d", url.Hostname(), 8443)
	}
	if url.Path != "/admin" {
		url.Path = "/admin"
	}
	return url.String(), nil
}

func (c *Config) CheckAuth() bool {
	if len(c.GetBearTokenHeaderValue()) < 10 {
		return false
	}
	if len(c.URL) < 1 {
		return false
	}
	if len(c.Provider) < 1 {
		return false
	}
	return c.ExpiredAtValid()
}

func (c *Config) ExpiredAtValid() bool {
	layout := "2006-01-02 15:04:05.999999999 -0700 MST"
	d, err := time.Parse(layout, c.ExpiresAt)
	if err != nil {
		return false
	}
	t1 := time.Now()
	return t1.Before(d)
}

func (c *Config) LoadCredentials() (*Credentials, error) {
	creds := &Credentials{}
	h, err := c.GetHost()
	if err != nil {
		return nil, err
	}
	if v, err := keyring.GetUsername(h); err == nil && len(v) > 0 {
		creds.Username = v
	}
	if v, err := keyring.GetPassword(h); err == nil && len(v) > 0 {
		creds.Password = v
	}

	return creds, nil
}

func (c *Config) StoreCredentials(crd *Credentials) error {
	h, err := c.GetHost()
	if err != nil {
		return err
	}
	if len(crd.Username) > 0 {
		if err := keyring.SetUsername(h, crd.Username); err != nil {
			return fmt.Errorf("could not store username in keychain %s", err)
		}
	}
	if len(crd.Password) > 0 {
		if err := keyring.SetPassword(h, crd.Password); err != nil {
			return fmt.Errorf("could not store password in keychain %s", err)
		}
	}

	return nil
}

func (c *Config) GetHost() (string, error) {
	if len(c.URL) == 0 {
		return "", ErrNoAddr
	}
	url, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}
	return url.Hostname(), nil
}
