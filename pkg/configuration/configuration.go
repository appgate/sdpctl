package configuration

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/keyring"
	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	URL                 string  `mapstructure:"url"`
	Provider            *string `mapstructure:"provider"`
	Insecure            bool    `mapstructure:"insecure"`
	Debug               bool    `mapstructure:"debug"`         // http debug flag
	Version             int     `mapstructure:"api_version"`   // api peer interface version
	BearerToken         *string `mapstructure:"bearer:squash"` // current logged in user token
	ExpiresAt           *string `mapstructure:"expires_at"`
	DeviceID            string  `mapstructure:"device_id"`
	PemFilePath         string  `mapstructure:"pem_filepath"` // deprecated in favor of pem_base64, kept for backwards compatibility
	PemBase64           *string `mapstructure:"pem_base64"`
	DisableVersionCheck bool    `mapstructure:"disable_version_check"`
	LastVersionCheck    string  `mapstructure:"last_version_check"`
	NoInteractive       bool    `mapstructure:"-"`
	CiMode              bool    `mapstructure:"-"`
	EventsPath          string  `mapstructure:"-"`
}

type Credentials struct {
	Username string
	Password string
}

func NewConfiguration(profile *string) (*Config, error) {
	dir := filesystem.ConfigDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create configuration directory: %w", err)
		}
	}
	if profiles.FileExists() {
		p, err := profiles.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read profiles from path '%s': %w", profiles.FilePath(), err)
		}
		var selectedProfile string
		if profile != nil && len(*profile) > 0 {
			selectedProfile = *profile
		} else if v := os.Getenv("SDPCTL_PROFILE"); len(v) > 0 {
			selectedProfile = v
		}
		if len(selectedProfile) > 0 {
			found := false
			for _, profile := range p.List {
				if selectedProfile == profile.Name {
					viper.AddConfigPath(profile.Directory)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid profile name: selected '%s', available %v", selectedProfile, p.Available())
			}
		} else if p.Current != nil {
			if profile, err := p.GetProfile(*p.Current); err == nil && profile != nil {
				// Move old logs if exists in profile dir
				matches, _ := filepath.Glob(profile.Directory + "/*.log")
				if len(matches) > 0 {
					if !profiles.LogDirectoryExists() {
						if err := profiles.CreateLogDirectory(); err != nil {
							logrus.WithError(err).Warn("failed to create log directory")
						}
					}
					for _, m := range matches {
						newPath := filepath.Join(filesystem.DataDir(), "logs", profile.Name+".log")
						if err := os.Rename(m, newPath); err != nil {
							logrus.WithError(err).Warn("failed to migrate old log file")
							profile.LogPath = m
						}
						profile.LogPath = newPath
					}
				}
				viper.AddConfigPath(profile.Directory)
			}
		} else if len(p.List) <= 0 {
			// There's a profile file, but there are no profiles configured or selected.
			// This probably only happens when config files are manually created, or some
			// configuration change has happened.
			// At this point, we fallback to creating the default profile and select that.
			pn := "default"
			p.Current = &pn
			defaultProfile, err := p.CreateDefaultProfile()
			if err != nil {
				os.Exit(1)
			}
			viper.AddConfigPath(defaultProfile.Directory)
		}
	} else {
		// if we don't have any profiles
		// we will assume there is only one Collective to respect
		// and we will default to base dir.
		viper.AddConfigPath(dir)

		// Migration code to move old root log file to proper place
		matches, _ := filepath.Glob(filesystem.ConfigDir() + "/*.log")
		matchOldLogs, _ := filepath.Glob(filesystem.DataDir() + "/*.log")
		matches = append(matches, matchOldLogs...)
		if len(matches) > 0 {
			logDir := filepath.Join(filesystem.DataDir(), "logs")
			if ok, err := util.FileExists(logDir); err == nil && !ok {
				os.MkdirAll(logDir, os.ModePerm)
			}
			logPath := filepath.Join(logDir, "sdpctl.log")
			for _, m := range matches {
				if err := os.Rename(m, logPath); err != nil {
					logrus.WithError(err).Warn("failed to migrate old log file")
				}
			}
		}

	}

	viper.SetConfigName("config")
	viper.SetEnvPrefix("SDPCTL")
	viper.AutomaticEnv()
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Its OK if we can't the file, fallback to arguments and/or environment variables
			// or configure it with sdpctl configure
		} else {
			return nil, fmt.Errorf("no sdpctl configuration found; run 'sdpctl configure'")
		}
	}
	return &Config{}, nil
}

func (c *Config) GetBearTokenHeaderValue() (string, error) {
	// if the bearer token is in the config, we assume the current environment does not support a keyring, so we will use it.
	// this will also include if the environment variable SDPCTL_BEARER is being used.
	if c.BearerToken != nil && len(*c.BearerToken) > 10 {
		return *c.BearerToken, nil
	}
	prefix, err := c.KeyringPrefix()
	if err != nil {
		return "", fmt.Errorf("Could not retrieve token for current host configuration %w", err)
	}
	v, err := keyring.GetBearer(prefix)
	if err != nil {
		return "", err
	}
	return v, nil
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

const SkipAuthCheck = "skipAuthCheck"

func IsAuthCheckEnabled(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return false
	}
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations[SkipAuthCheck] == "true" {
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

const NeedUpdateAPIConfig = "updateAPIConfig"

func NeedUpdatedAPIVersionConfig(cmd *cobra.Command) bool {
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations[NeedUpdateAPIConfig] == "true" {
			return true
		}
	}
	return false
}

var ErrNoAddr = errors.New("No valid address set, run 'sdpctl configure' or set SDPCTL_URL")

func NormalizeConfigurationURL(u string) (string, error) {
	if len(u) <= 0 {
		return "", ErrNoAddr
	}
	url, err := util.NormalizeURL(u)
	if err != nil {
		return "", err
	}
	if len(url.Port()) <= 0 {
		url.Host = fmt.Sprintf("%s:%d", url.Hostname(), 8443)
	}
	if url.Path != "/admin" {
		url.Path = "/admin"
	}
	return url.String(), nil
}

func (c *Config) IsRequireAuthentication() bool {
	if len(c.URL) < 1 {
		return true
	}
	if c.Provider == nil {
		return true
	}
	t, err := c.GetBearTokenHeaderValue()
	if err != nil {
		return true
	}
	if len(t) < 10 {
		return true
	}
	return !c.ExpiredAtValid()
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

func (c *Config) CheckForUpdate(out io.Writer, client *http.Client, current string) (*Config, error) {
	// Check if version check is disabled in configuration
	if c.DisableVersionCheck {
		return c, cmdutil.ErrVersionCheckDisabled
	}

	// Check if version check has already been done today
	lastCheck, err := time.Parse(time.RFC3339Nano, c.LastVersionCheck)
	if err == nil {
		yesterday := time.Now().AddDate(0, 0, -1)
		if !lastCheck.Before(yesterday) {
			return c, cmdutil.ErrDailyVersionCheck
		}
	}

	// Perform version check
	const cliReleasesURL = "https://api.github.com/repos/appgate/sdpctl/releases"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cliReleasesURL, nil)
	if err != nil {
		return c, err
	}

	// Write new check time to config after request is made
	c.LastVersionCheck = time.Now().Format(time.RFC3339Nano)
	viper.Set("last_version_check", c.LastVersionCheck)
	req.Header.Add("Accept", "application/vnd.github+json")
	res, err := api.RequestRetry(client, req)
	if err != nil {
		return c, err
	}
	defer res.Body.Close()

	type githubRelease struct {
		TagName    string `json:"tag_name"`
		PreRelease bool   `json:"pre_release"`
		Draft      bool   `json:"draft"`
	}

	type releaseList []githubRelease

	if res.StatusCode != http.StatusOK {
		return c, fmt.Errorf("unexpected request status: %d", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return c, err
	}

	var releases releaseList
	if err := json.Unmarshal(b, &releases); err != nil {
		return c, err
	}

	v, err := version.NewVersion(current)
	if err != nil {
		return c, err
	}

	var latest *version.Version
	r := releases[0]
	n, err := version.NewVersion(r.TagName)
	if err != nil {
		return c, err
	}
	if !r.Draft && !r.PreRelease && n.GreaterThan(v) {
		latest = n
	}
	if latest != nil {
		fmt.Fprintf(out, "NOTICE: A new version of sdpctl is available for download: %s\nDownload it here: https://github.com/appgate/sdpctl/releases/tag/%s\n\n", latest.Original(), latest.Original())
	}
	return c, nil
}

func ReadPemFile(path string) (*x509.Certificate, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Path %s does not exist", path)
		}
		return nil, fmt.Errorf("%s - %s", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path %s is a directory, not a file", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("not a file %s %s", path, err)
	}
	pemData, certBytes := pem.Decode(b)
	if pemData != nil {
		certBytes = pemData.Bytes
	}

	// See if we can parse the certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return cert, nil
}

func CertificateDetails(cert *x509.Certificate) string {
	var sb strings.Builder

	if len(cert.Subject.CommonName) > 0 {
		sb.WriteString(fmt.Sprintf("[Subject]\n\t%s\n", cert.Subject.CommonName))
	}
	if len(cert.Issuer.CommonName) > 0 {
		sb.WriteString(fmt.Sprintf("[Issuer]\n\t%s\n", cert.Issuer.CommonName))
	}
	if cert.SerialNumber != nil {
		sb.WriteString(fmt.Sprintf("[Serial Number]\n\t%s\n", cert.SerialNumber))
	}

	sb.WriteString(fmt.Sprintf("[Not Before]\n\t%s\n", cert.NotBefore))
	sb.WriteString(fmt.Sprintf("[Not After]\n\t%s\n", cert.NotAfter))

	var sha1buf strings.Builder
	for i, f := range sha1.Sum(cert.Raw) {
		if i > 0 {
			sha1buf.Write([]byte(":"))
		}
		sha1buf.Write([]byte(fmt.Sprintf("%02X", f)))
	}
	sb.WriteString(fmt.Sprintf("[Thumbprint SHA-1]\n\t%s\n", sha1buf.String()))

	var sha256buf strings.Builder
	for i, f := range sha256.Sum256(cert.Raw) {
		if i > 0 {
			sha256buf.Write([]byte(":"))
		}
		sha256buf.Write([]byte(fmt.Sprintf("%02X", f)))
	}
	sb.WriteString(fmt.Sprintf("[Thumbprint SHA-256]\n\t%s\n", sha256buf.String()))

	return sb.String()
}
