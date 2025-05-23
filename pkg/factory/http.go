package factory

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/device"
	"github.com/appgate/sdpctl/pkg/serviceusers"
	"golang.org/x/net/http/httpproxy"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/util"
)

type Factory struct {
	// HTTPClient is the underlying HTTP client used in APIClient and CustomHTTPClient
	HTTPClient func() (*http.Client, error)
	// CustomHTTPClient includes a custom HTTP.Client that includes the default
	// headers to integrate with a Controller
	// the custom HTTP client includes the default transport layer to import TLS certificate
	// and applies Accept, Authorization, and User-Agent headers to all requests
	CustomHTTPClient func() (*http.Client, error)
	// HTTPTransport is used by all HTTP Clients to import custom TLS certificate and set timeout values
	HTTPTransport func() (*http.Transport, error)
	// APIClient is the generated api client based on the openapi-generator https://github.com/appgate/sdp-api-client-go
	APIClient      func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance      func(c *configuration.Config) (*appliance.Appliance, error)
	Device         func(c *configuration.Config) (*device.Device, error)
	ServiceUsers   func(c *configuration.Config) (*serviceusers.ServiceUsersAPI, error)
	DockerRegistry func(s string) (*url.URL, error)
	BaseURL        func() string
	userAgent      string
	Config         *configuration.Config
	IOOutWriter    io.Writer
	Stdin          io.ReadCloser
	StdErr         io.Writer
	SpinnerOut     io.Writer
	neverPrompt    bool
}

// Set on build time
var dockerRegistry string

func New(appVersion string, config *configuration.Config) *Factory {
	f := &Factory{}
	f.Config = config
	f.userAgent = "sdpctl/" + appVersion
	f.HTTPTransport = httpTransport(f)       // depends on config
	f.HTTPClient = httpClientFunc(f)         // depends on config
	f.CustomHTTPClient = customHTTPClient(f) // depends on config
	f.APIClient = apiClientFunc(f)           // depends on config
	f.Appliance = applianceFunc(f)           // depends on config
	f.Device = deviceFunc(f)                 // depends on config
	f.ServiceUsers = serviceUsersFunc(f)     // depends on config
	f.BaseURL = BaseURL(f)
	f.IOOutWriter = os.Stdout
	f.Stdin = os.Stdin
	f.StdErr = os.Stderr
	f.SpinnerOut = os.Stdout
	f.DockerRegistry = f.GetDockerRegistry

	return f
}

func (f *Factory) DisablePrompt(v bool) {
	f.neverPrompt = v
}

func (f *Factory) CanPrompt() bool {
	if f.neverPrompt {
		return false
	}
	return cmdutil.IsTTYRead(f.Stdin) && cmdutil.IsTTY(f.StdErr)
}

func (f *Factory) SetSpinnerOutput(o io.Writer) {
	f.SpinnerOut = o
}

func (f *Factory) GetSpinnerOutput() func() io.Writer {
	return func() io.Writer {
		return f.SpinnerOut
	}
}

func BaseURL(f *Factory) func() string {
	return func() string {
		url, err := configuration.NormalizeConfigurationURL(f.Config.URL)
		if err != nil {
			return f.Config.URL
		}
		return url
	}
}

// GetDockerRegistry parses and returns a normalized URL for the docker registry to be used
// The URL needs to be valid and the following priority will be evaluated
// 1. Argument 's'
// 2. If the 'SDPCTL_DOCKER_REGISTRY' environment variable is set
// 3. The default registry which is set during build time
func (f *Factory) GetDockerRegistry(s string) (*url.URL, error) {
	reg := util.Getenv("SDPCTL_DOCKER_REGISTRY", dockerRegistry)
	if len(s) > 0 {
		reg = s
	}
	return util.NormalizeURL(reg)
}

var proxyFunc func(*url.URL) (*url.URL, error)

func proxyFromEnvironment(req *http.Request) (*url.URL, error) {
	if proxyFunc == nil {
		proxyFunc = httpproxy.FromEnvironment().ProxyFunc()
	}
	return proxyFunc(req.URL)
}

func httpTransport(f *Factory) func() (*http.Transport, error) {
	return func() (*http.Transport, error) {
		cfg := f.Config
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		if cfg.PemBase64 != nil {
			data, err := base64.StdEncoding.DecodeString(*cfg.PemBase64)
			if err != nil {
				return nil, fmt.Errorf("could not decode stored certificate %w", err)
			}
			cert, err := x509.ParseCertificate(data)
			if err != nil {
				return nil, fmt.Errorf("could not parse certificate %w", err)
			}
			rootCAs.AddCert(cert)
		} else if ok, err := util.FileExists(cfg.PemFilePath); err == nil && ok {
			// deprecated: TODO remove in future version
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
			Proxy: proxyFromEnvironment,
		}

		return tr, nil
	}
}

type customTransport struct {
	token, accept, useragent string
	underlyingTransport      http.RoundTripper
}

func (ct *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", ct.token)
	req.Header.Add("User-Agent", ct.useragent)
	req.Header.Add("Accept", ct.accept)

	// overwrite Accept header if we have anything in the context
	if accept, ok := req.Context().Value(api.ContextAcceptValue).(string); ok {
		req.Header.Set("Accept", accept)
	}

	return ct.underlyingTransport.RoundTrip(req)
}

func customHTTPClient(f *Factory) func() (*http.Client, error) {
	return func() (*http.Client, error) {
		cfg := f.Config
		client, err := f.HTTPClient()
		if err != nil {
			return nil, err
		}
		parentTransport, err := f.HTTPTransport()
		if err != nil {
			return nil, err
		}
		token, err := cfg.GetBearTokenHeaderValue()
		if err != nil {
			return nil, err
		}
		client.Transport = &customTransport{
			token:               fmt.Sprintf("Bearer %s", token),
			accept:              fmt.Sprintf("application/vnd.appgate.peer-v%d+json", cfg.Version),
			useragent:           f.userAgent,
			underlyingTransport: parentTransport,
		}
		return client, nil
	}
}

func httpClientFunc(f *Factory) func() (*http.Client, error) {
	return func() (*http.Client, error) {
		tr, err := f.HTTPTransport()
		if err != nil {
			return nil, err
		}
		c := &http.Client{
			Transport: tr,
		}
		return c, nil
	}
}

func apiClientFunc(f *Factory) func(c *configuration.Config) (*openapi.APIClient, error) {
	return func(cfg *configuration.Config) (*openapi.APIClient, error) {
		hc, err := f.HTTPClient()
		if err != nil {
			return nil, err
		}

		cfg.URL, err = configuration.NormalizeConfigurationURL(cfg.URL)
		if err != nil {
			return nil, err
		}
		clientCfg := &openapi.Configuration{
			DefaultHeader: map[string]string{
				"Accept": fmt.Sprintf("application/vnd.appgate.peer-v%d+json", cfg.Version),
			},
			Debug:     cfg.Debug,
			UserAgent: f.userAgent,
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

func getClients(f *Factory, cfg *configuration.Config) (*http.Client, *openapi.APIClient, error) {
	httpClient, err := f.CustomHTTPClient()
	if err != nil {
		return nil, nil, err
	}
	apiClient, err := f.APIClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	return httpClient, apiClient, nil
}

func applianceFunc(f *Factory) func(c *configuration.Config) (*appliance.Appliance, error) {
	return func(cfg *configuration.Config) (*appliance.Appliance, error) {
		httpClient, apiClient, err := getClients(f, cfg)
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

func deviceFunc(f *Factory) func(c *configuration.Config) (*device.Device, error) {
	return func(cfg *configuration.Config) (*device.Device, error) {
		httpClient, apiClient, err := getClients(f, cfg)
		if err != nil {
			return nil, err
		}
		bearerToken, err := cfg.GetBearTokenHeaderValue()
		if err != nil {
			return nil, err
		}
		t := &device.Device{
			HTTPClient: httpClient,
			APIClient:  apiClient,
			Token:      bearerToken,
		}
		return t, nil
	}
}

func serviceUsersFunc(f *Factory) func(c *configuration.Config) (*serviceusers.ServiceUsersAPI, error) {
	return func(cfg *configuration.Config) (*serviceusers.ServiceUsersAPI, error) {
		_, apiClient, err := getClients(f, cfg)
		if err != nil {
			return nil, err
		}
		bearerToken, err := cfg.GetBearTokenHeaderValue()
		if err != nil {
			return nil, err
		}
		return serviceusers.NewServiceUsersAPI(apiClient.ServiceUsersApi, bearerToken), nil
	}
}
