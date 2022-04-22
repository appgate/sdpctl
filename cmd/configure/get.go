package configure

import (
	"fmt"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/hashcode"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewConfigGetCmd return a new config command
func NewConfigGetCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use: "get",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		Short:  "Print current config to stdout",
		Long:   "Show computed config based on environment variables and config file (username and password is excluded)",
		Hidden: true,
		RunE: func(c *cobra.Command, args []string) error {
			out := f.IOOutWriter
			stderr := f.StdErr

			fmt.Fprintf(out, "\nComputed Configuration:\n")
			addr, err := configuration.NormalizeURL(f.Config.URL)
			if err != nil {
				fmt.Fprintf(stderr, "[ERROR] NormalizeURL %s", err)
			}
			fmt.Fprintf(out, "NormalizeURL: %s\n", addr)
			for k, v := range viper.GetViper().AllSettings() {
				fmt.Fprintf(out, "%s: %+v\n", k, v)
			}

			h, err := f.Config.GetHost()
			if err != nil {
				fmt.Fprintf(stderr, "[ERROR] GetHost %s", err)
			}
			fmt.Fprintf(out, "Host: %s\n", h)
			fmt.Fprintf(out, "Host hashcode: %d\n", hashcode.String(h))

			return nil
		},
	}
}
