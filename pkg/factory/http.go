package factory

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

type Factory struct {
	HTTPClient  func() (*http.Client, error)
	APIClient   func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance   func(c *configuration.Config) (*appliance.Appliance, error)
	Config      *configuration.Config
	IOOutWriter io.Writer
	Stdin       io.Reader
	StdErr      io.Reader
}

func New(appVersion string, config *configuration.Config) *Factory {
	f := &Factory{}
	f.Config = config
	f.HTTPClient = httpClientFunc(f)           // depends on config
	f.APIClient = apiClientFunc(f, appVersion) // depends on config
	f.Appliance = applianceFunc(f, appVersion) // depends on config
	f.IOOutWriter = os.Stdout
	f.Stdin = os.Stdin
	return f
}

func httpClientFunc(f *Factory) func() (*http.Client, error) {
	return func() (*http.Client, error) {
		cfg := f.Config
		timeout := 300
		timeoutDuration := time.Duration(timeout)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Insecure,
			},
			Dial: (&net.Dialer{
				Timeout: timeoutDuration * time.Second,
			}).Dial,
			TLSHandshakeTimeout: timeoutDuration * time.Second,
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
		clientCfg := &openapi.Configuration{
			DefaultHeader: map[string]string{
				"Accept": fmt.Sprintf("application/vnd.appgate.peer-v%d+json", cfg.Version),
			},
			Debug:     cfg.Debug,
			UserAgent: "appgatectl/" + appVersion + "/go",
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

func applianceFunc(f *Factory, appVersion string) func(c *configuration.Config) (*appliance.Appliance, error) {
	return func(cfg *configuration.Config) (*appliance.Appliance, error) {
		hc, err := f.HTTPClient()
		if err != nil {
			return nil, err
		}
		c, err := f.APIClient(cfg)
		if err != nil {
			return nil, err
		}
		a := &appliance.Appliance{
			APIClient:  c,
			HTTPClient: hc,
			Token:      cfg.GetBearTokenHeaderValue(),
		}
		return a, nil
	}
}
