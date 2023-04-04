package factory

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/serviceusers"
	"github.com/appgate/sdpctl/pkg/token"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
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
	// BasicAuthClient is used when communicating with third party API:s that need basic authentication
	BasicAuthClient func(username, password string) (*http.Client, error)
	// HTTPTransport is used by all HTTP Clients to import custom TLS certificate and set timeout values
	HTTPTransport func() (*http.Transport, error)
	// APIClient is the generated api client based on the openapi-generator https://github.com/appgate/sdp-api-client-go
	APIClient    func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance    func(c *configuration.Config) (*appliance.Appliance, error)
	Token        func(c *configuration.Config) (*token.Token, error)
	ServiceUsers func(c *configuration.Config) (*serviceusers.ServiceUsersAPI, error)
	userAgent    string
	Config       *configuration.Config
	IOOutWriter  io.Writer
	Stdin        io.ReadCloser
	StdErr       io.Writer
	SpinnerOut   io.Writer
	neverPrompt  bool
}

func New(appVersion string, config *configuration.Config) *Factory {
	f := &Factory{}
	f.Config = config
	f.userAgent = "sdpctl/" + appVersion
	f.HTTPTransport = httpTransport(f)       // depends on config
	f.HTTPClient = httpClientFunc(f)         // depends on config
	f.CustomHTTPClient = customHTTPClient(f) // depends on config
	f.BasicAuthClient = basicAuthClient(f)   // depends on config
	f.APIClient = apiClientFunc(f)           // depends on config
	f.Appliance = applianceFunc(f)           // depends on config
	f.Token = tokenFunc(f)                   // depends on config
	f.ServiceUsers = serviceUsersFunc(f)     // depends on config
	f.IOOutWriter = os.Stdout
	f.Stdin = os.Stdin
	f.StdErr = os.Stderr
	f.SpinnerOut = os.Stdout

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

func httpTransport(f *Factory) func() (*http.Transport, error) {
	return func() (*http.Transport, error) {
		cfg := f.Config
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
		}
		if key, ok := os.LookupEnv("HTTP_PROXY"); ok {
			proxyURL, err := url.Parse(key)
			if err != nil {
				return nil, err
			}
			tr.Proxy = http.ProxyURL(proxyURL)
		}
		return tr, nil
	}
}

type customTransport struct {
	token, accept, useragent string
	underlyingTransport      http.RoundTripper
}

type ContextKey string

const ContextAcceptValue ContextKey = "Accept"

func (ct *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", ct.token)
	req.Header.Add("User-Agent", ct.useragent)
	req.Header.Add("Accept", ct.accept)

	// overwrite Accept header if we have anything in the context
	if accept, ok := req.Context().Value(ContextAcceptValue).(string); ok {
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
			token:               token,
			accept:              fmt.Sprintf("application/vnd.appgate.peer-v%d+json", cfg.Version),
			useragent:           f.userAgent,
			underlyingTransport: parentTransport,
		}
		return client, nil
	}
}

type basicAuthTransport struct {
	username, password, useragent, accept string
	underlyingTransport                   http.RoundTripper
}

func (bat basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", bat.useragent)
	req.Header.Add("Accept", bat.accept)
	req.SetBasicAuth(bat.username, bat.password)

	// overwrite Accept header if we have anything in the context
	if accept, ok := req.Context().Value(ContextAcceptValue).(string); ok {
		req.Header.Set("Accept", accept)
	}

	return bat.underlyingTransport.RoundTrip(req)
}

func basicAuthClient(f *Factory) func(username, password string) (*http.Client, error) {
	return func(username, password string) (*http.Client, error) {
		client, err := f.HTTPClient()
		if err != nil {
			return nil, err
		}
		parentTransport, err := f.HTTPTransport()
		if err != nil {
			return nil, err
		}
		if u := os.Getenv("SDPCTL_DOCKER_REGISTRY_USERNAME"); len(u) > 0 {
			username = u
		}
		if p := os.Getenv("SDPCTL_DOCKER_REGISTRY_PASSWORD"); len(p) > 0 {
			password = p
		}
		client.Transport = basicAuthTransport{
			underlyingTransport: parentTransport,
			username:            username,
			password:            password,
			useragent:           f.userAgent,
			accept:              fmt.Sprintf("application/vnd.appgate.peer-v%d+json", f.Config.Version),
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

		cfg.URL, err = configuration.NormalizeURL(cfg.URL)
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

func tokenFunc(f *Factory) func(c *configuration.Config) (*token.Token, error) {
	return func(cfg *configuration.Config) (*token.Token, error) {
		httpClient, apiClient, err := getClients(f, cfg)
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
