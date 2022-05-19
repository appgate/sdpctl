package factory

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/appgate/sdpctl/pkg/token"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/util"
)

type Factory struct {
	HTTPClient  func() (*http.Client, error)
	APIClient   func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance   func(c *configuration.Config) (*appliance.Appliance, error)
	Token       func(c *configuration.Config) (*token.Token, error)
	Config      *configuration.Config
	IOOutWriter io.Writer
	Stdin       io.Reader
	StdErr      io.Writer
	SpinnerOut  io.Writer
}

func New(appVersion string, config *configuration.Config) *Factory {
	f := &Factory{}
	f.Config = config
	f.HTTPClient = httpClientFunc(f)           // depends on config
	f.APIClient = apiClientFunc(f, appVersion) // depends on config
	f.Appliance = applianceFunc(f, appVersion) // depends on config
	f.Token = tokenFunc(f, appVersion)         // depends on config
	f.IOOutWriter = os.Stdout
	f.Stdin = os.Stdin
	f.StdErr = os.Stderr
	f.SpinnerOut = os.Stdout

	return f
}

func (f *Factory) SetSpinnerOutput(o io.Writer) {
	f.SpinnerOut = o
}
func (f *Factory) GetSpinnerOutput() func() io.Writer {
	return func() io.Writer {
		return f.SpinnerOut
	}
}
func httpClientFunc(f *Factory) func() (*http.Client, error) {
	return func() (*http.Client, error) {
		cfg := f.Config
		timeout := 5
		if cfg.Timeout > timeout {
			timeout = cfg.Timeout
		}

		timeoutDuration := time.Duration(timeout)

		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		if ok, err := util.FileExists(cfg.PemFilePath); err == nil && ok {
			certs, err := os.ReadFile(cfg.PemFilePath)
			if err != nil {
				return nil, err
			}
			if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
				return nil, fmt.Errorf("unable to append cert %s", cfg.PemFilePath)
			}
		}
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Insecure,
				RootCAs:            rootCAs,
			},
			Dial: (&net.Dialer{
				Timeout: timeoutDuration * time.Second,
			}).Dial,
			TLSHandshakeTimeout: timeoutDuration * time.Second,
		}
		if key, ok := os.LookupEnv("HTTP_PROXY"); ok {
			proxyURL, err := url.Parse(key)
			if err != nil {
				return nil, err
			}
			tr.Proxy = http.ProxyURL(proxyURL)
		}
		c := &http.Client{
			Transport: tr,
			Timeout:   ((timeoutDuration * 2) * time.Second),
		}
		return c, nil
	}
}

func apiClientFunc(f *Factory, appVersion string) func(c *configuration.Config) (*openapi.APIClient, error) {
	return func(cfg *configuration.Config) (*openapi.APIClient, error) {
		hc, err := f.HTTPClient()
		if err != nil {
			return nil, err
		}

		cfg.URL, err = configuration.NormalizeURL(cfg.URL)
		if err != nil {
			return nil, err
		}
		clientCfg := &openapi.Configuration{
			DefaultHeader: map[string]string{
				"Accept": fmt.Sprintf("application/vnd.appgate.peer-v%d+json", cfg.Version),
			},
			Debug:     cfg.Debug,
			UserAgent: "sdpctl/" + appVersion + "/go",
			Servers: []openapi.ServerConfiguration{
				{
					URL: cfg.URL,
				},
			},
			HTTPClient: hc,
		}

		return openapi.NewAPIClient(clientCfg), nil
	}
}

func getClients(f *Factory, appVersion string, cfg *configuration.Config) (*http.Client, *openapi.APIClient, error) {
	httpClient, err := f.HTTPClient()
	if err != nil {
		return nil, nil, err
	}
	apiClient, err := f.APIClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	return httpClient, apiClient, nil
}

func applianceFunc(f *Factory, appVersion string) func(c *configuration.Config) (*appliance.Appliance, error) {
	return func(cfg *configuration.Config) (*appliance.Appliance, error) {
		httpClient, apiClient, err := getClients(f, appVersion, cfg)
		if err != nil {
			return nil, err
		}
		token, err := cfg.GetBearTokenHeaderValue()
		if err != nil {
			return nil, err
		}
		a := &appliance.Appliance{
			HTTPClient: httpClient,
			APIClient:  apiClient,
			Token:      token,
		}
		return a, nil
	}
}

func tokenFunc(f *Factory, appVersion string) func(c *configuration.Config) (*token.Token, error) {
	return func(cfg *configuration.Config) (*token.Token, error) {
		httpClient, apiClient, err := getClients(f, appVersion, cfg)
		if err != nil {
			return nil, err
		}
		bearerToken, err := cfg.GetBearTokenHeaderValue()
		if err != nil {
			return nil, err
		}
		t := &token.Token{
			HTTPClient: httpClient,
			APIClient:  apiClient,
			Token:      bearerToken,
		}
		return t, nil
	}
}
