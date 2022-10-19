package license

import (
	"fmt"
	"io"
	"net/http"

	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

type licenseOpts struct {
	Out        io.Writer
	BaseURL    string
	HTTPClient func() (*http.Client, error)
}

type customTransport struct {
	token, accept, useragent string
	underlyingTransport      http.RoundTripper
}

func (ct *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Accept", ct.accept)
	req.Header.Add("Authorization", ct.token)
	req.Header.Add("User-Agent", ct.useragent)

	return ct.underlyingTransport.RoundTrip(req)
}

// NewLicenseCmd return a new license subcommand
func NewLicenseCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "license",
		TraverseChildren: true,
	}

	cfg := f.Config
	opts := &licenseOpts{
		Out:     f.IOOutWriter,
		BaseURL: cfg.URL,
		HTTPClient: func() (*http.Client, error) {
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
				useragent:           f.UserAgent,
				underlyingTransport: parentTransport,
			}
			return client, nil
		},
	}
	cmd.AddCommand(NewPruneCmd(opts))

	return cmd
}
