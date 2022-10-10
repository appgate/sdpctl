package configuration

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/appgate/sdpctl/pkg/keyring"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	URL         string `mapstructure:"url"`
	Provider    string `mapstructure:"provider"`
	Insecure    bool   `mapstructure:"insecure"`
	Debug       bool   `mapstructure:"debug"`       // http debug flag
	Version     int    `mapstructure:"api_version"` // api peer interface version
	BearerToken string `mapstructure:"bearer"`      // current logged in user token
	ExpiresAt   string `mapstructure:"expires_at"`
	DeviceID    string `mapstructure:"device_id"`
	PemFilePath string `mapstructure:"pem_filepath"`
	Timeout     int    // HTTP timeout, not supported in the config file.
}

type Credentials struct {
	Username string
	Password string
}

func (c *Config) GetBearTokenHeaderValue() (string, error) {
	// if the bearer token is in the config, we assume the current environment does not support a keyring, so we will use it.
	// this will also include if the environment variable SDPCTL_BEARER is being used.
	if len(c.BearerToken) > 10 {
		return fmt.Sprintf("Bearer %s", c.BearerToken), nil
	}
	h, err := c.GetHost()
	if err != nil {
		return "", fmt.Errorf("could not retrieve token for current host configuration %w", err)
	}
	v, err := keyring.GetBearer(h)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Bearer %s", v), nil
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

func CheckMinAPIVersionRestriction(cmd *cobra.Command, currentVersion int) error {
	c := cmd
	for c != nil {
		if c.Annotations != nil {
			if s, ok := c.Annotations["MinAPIVersion"]; ok {
				if minVersion, err := strconv.Atoi(s); err == nil && currentVersion < minVersion {
					return fmt.Errorf("Minimum API version %d is required to use the '%s' command. Current API version is %d", minVersion, c.Name(), currentVersion)
				}
			}
		}
		c = c.Parent()
	}
	return nil
}

func NeedUpdatedAPIVersionConfig(cmd *cobra.Command) bool {
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations["updateAPIConfig"] == "true" {
			return true
		}
	}
	return false
}

var ErrNoAddr = errors.New("no valid address set, run 'sdpctl configure' or set SDPCTL_URL")

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
	if len(c.URL) < 1 {
		return false
	}
	if len(c.Provider) < 1 {
		return false
	}
	t, err := c.GetBearTokenHeaderValue()
	if err != nil {
		return false
	}
	if len(t) < 10 {
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
	return t1.Add(-time.Hour * 2).Before(d)
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

func (c *Config) ClearCredentials() error {
	h, err := c.GetHost()
	if err != nil {
		return err
	}
	if err := keyring.ClearCredentials(h); err != nil {
		return err
	}
	c.BearerToken = ""
	c.ExpiresAt = ""
	c.Provider = ""
	keys := []string{"bearer", "expires_at", "provider"}
	allKeys := viper.AllKeys()
	for _, k := range keys {
		if util.InSlice(k, allKeys) {
			viper.Set(k, "")
		}
	}
	if err := viper.WriteConfig(); err != nil {
		// only return error if there is a config file to write to
		// as is the case when using environment variables for configuration
		// or in testing
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return err
		}
	}
	return nil
}

func (c *Config) ClearBearer() error {
	h, err := c.GetHost()
	if err != nil {
		return err
	}
	if err := keyring.DeleteBearer(h); err != nil {
		return err
	}
	viper.Set("bearer", "")
	viper.Set("expires_at", "")
	return nil
}

func (c *Config) StoreCredentials(username, password string) error {
	h, err := c.GetHost()
	if err != nil {
		return err
	}
	if err := keyring.SetUsername(h, username); err != nil {
		return fmt.Errorf("could not store username in keychain %w", err)
	}
	if err := keyring.SetPassword(h, password); err != nil {
		return fmt.Errorf("could not store password in keychain %w", err)
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
