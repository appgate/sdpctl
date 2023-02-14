package configuration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/appgate/sdpctl/pkg/keyring"
	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

type Meta struct {
	DisableVersionCheck bool   `mapstructure:"disable_version_check" json:"disable_version_check"`
	LastChecked         string `mapstructure:"last_checked" json:"last_checked"`
}

type Config struct {
	URL         string  `mapstructure:"url"`
	Provider    *string `mapstructure:"provider"`
	Insecure    bool    `mapstructure:"insecure"`
	Debug       bool    `mapstructure:"debug"`         // http debug flag
	Version     int     `mapstructure:"api_version"`   // api peer interface version
	BearerToken *string `mapstructure:"bearer:squash"` // current logged in user token
	ExpiresAt   *string `mapstructure:"expires_at"`
	DeviceID    string  `mapstructure:"device_id"`
	PemFilePath string  `mapstructure:"pem_filepath"`
	Timeout     int     // HTTP timeout, not supported in the config file.
	Meta        Meta    `mapstructure:"meta"`
}

type Credentials struct {
	Username string
	Password string
}

func (c *Config) GetBearTokenHeaderValue() (string, error) {
	// if the bearer token is in the config, we assume the current environment does not support a keyring, so we will use it.
	// this will also include if the environment variable SDPCTL_BEARER is being used.
	if c.BearerToken != nil && len(*c.BearerToken) > 10 {
		return fmt.Sprintf("Bearer %s", *c.BearerToken), nil
	}
	prefix, err := c.KeyringPrefix()
	if err != nil {
		return "", fmt.Errorf("Could not retrieve token for current host configuration %w", err)
	}
	v, err := keyring.GetBearer(prefix)
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
					if message, ok := c.Annotations["ErrorMessage"]; ok {
						return errors.New(message)
					}
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

var ErrNoAddr = errors.New("No valid address set, run 'sdpctl configure' or set SDPCTL_URL")

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
	if c.Provider == nil {
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
	if c.ExpiresAt == nil {
		return false
	}
	layout := "2006-01-02 15:04:05.999999999 -0700 MST"
	t1, err := time.Parse(layout, *c.ExpiresAt)
	if err != nil {
		return false
	}
	now := time.Now().Add(-time.Hour * 2)
	return t1.After(now)
}

func (c *Config) LoadCredentials() (*Credentials, error) {
	creds := &Credentials{}
	prefix, err := c.KeyringPrefix()
	if err != nil {
		return nil, err
	}
	if v, err := keyring.GetUsername(prefix); err == nil && len(v) > 0 {
		creds.Username = v
	}
	if v, err := keyring.GetPassword(prefix); err == nil && len(v) > 0 {
		creds.Password = v
	}

	return creds, nil
}

func (c *Config) ClearCredentials() error {
	prefix, err := c.KeyringPrefix()
	if err != nil {
		return err
	}
	if err := keyring.ClearCredentials(prefix); err != nil {
		return err
	}
	if err := c.ClearBearer(); err != nil {
		return err
	}
	c.Provider = nil
	keys := []string{"expires_at", "provider"}
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
	c.BearerToken = nil
	c.ExpiresAt = nil

	return nil
}

func (c *Config) StoreCredentials(username, password string) error {
	prefix, err := c.KeyringPrefix()
	if err != nil {
		return err
	}
	if err := keyring.SetUsername(prefix, username); err != nil {
		return fmt.Errorf("Could not store username in keychain %w", err)
	}
	if err := keyring.SetPassword(prefix, password); err != nil {
		return fmt.Errorf("Could not store password in keychain %w", err)
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

// KeyringPrefix is the raw string values that will be used in the keyring package
// it needs to be a unique, reproducible value for the selected Collective + profile.
// Downstream, this string value will be converted to a integer value (pkg/hashcode)
// and used as a prefix when storing values in the keyring/keychain.
func (c *Config) KeyringPrefix() (string, error) {
	h, err := c.GetHost()
	if err != nil {
		return "", err
	}
	p, err := profiles.Read()
	if err == nil {
		if p.CurrentExists() {
			if c, err := p.CurrentProfile(); err == nil {
				return c.Name + h, nil
			}
		}
	}
	return h, nil
}

func (c *Config) CheckForUpdate(out io.Writer, current string) (Meta, error) {
	// Check if version check is disabled in configuration
	if c.Meta.DisableVersionCheck {
		return c.Meta, errors.New("version check disabled")
	}

	// Check if version check has already been done today
	lastCheck, err := time.Parse(time.RFC3339Nano, c.Meta.LastChecked)
	if err == nil {
        yesterday := time.Now().AddDate(0, 0, -1)
		if !lastCheck.Before(yesterday) {
			return c.Meta, errors.New("version check already done today")
		}
	}

	// Perform version check
	const cliReleasesURL = "https://api.github.com/repos/appgate/sdpctl/releases"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cliReleasesURL, nil)
	// Write new check time to config after request is made
	c.Meta.LastChecked = time.Now().Format(time.RFC3339Nano)
	if err != nil {
		return c.Meta, err
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return c.Meta, err
	}
	defer res.Body.Close()

	type githubRelease struct {
		TagName    string `json:"tag_name"`
		PreRelease bool   `json:"pre_release"`
		Draft      bool   `json:"draft"`
	}

	type releaseList []githubRelease

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return c.Meta, err
	}

	var releases releaseList
	if err := json.Unmarshal(b, &releases); err != nil {
		return c.Meta, err
	}

	v, err := version.NewVersion(current)
	if err != nil {
		return c.Meta, err
	}

	var latest *version.Version
	r := releases[0]
	n, err := version.NewVersion(r.TagName)
	if err != nil {
		return c.Meta, err
	}
	if !r.Draft && !r.PreRelease && n.GreaterThan(v) {
		latest = n
	}
	if latest == nil {
		return c.Meta, errors.New("already at latest version")
	}
	fmt.Fprintf(out, "NOTICE: A new version of sdpctl is available for download: %s\nDownload it here: https://github.com/appgate/sdpctl/releases/tag/%s\n\n", latest.Original(), latest.Original())
	return c.Meta, nil
}
