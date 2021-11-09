package factory

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

type Factory struct {
	HTTPClient  func() (*http.Client, error)
	APIClient   func(c *config.Config) (*openapi.APIClient, error)
	Config      *config.Config
	IOOutWriter io.Writer
}

func New(appVersion string, config *config.Config) *Factory {
	f := &Factory{}
	f.Config = config
	f.HTTPClient = httpClientFunc(f)           // depends on config
	f.APIClient = apiClientFunc(f, appVersion) // depends on config
	f.IOOutWriter = os.Stdout
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

func apiClientFunc(f *Factory, appVersion string) func(c *config.Config) (*openapi.APIClient, error) {
	return func(cfg *config.Config) (*openapi.APIClient, error) {
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
