package cmd

import (
	"fmt"
	"io"
	"net/url"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewOpenCmd return a new open command
func NewOpenCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use: "open",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		Short: "Open the web UI in your default browser",
		RunE: func(c *cobra.Command, args []string) error {
			addr, err := configuration.NormalizeURL(f.Config.URL)
			if err != nil {
				return fmt.Errorf("could not normalize addr %w", err)
			}
			webUI, err := url.Parse(addr)
			if err != nil {
				return fmt.Errorf("could not parse addr %w", err)
			}
			browser.Stderr = io.Discard
			webUI.Path = "/ui"
			if err := browser.OpenURL(webUI.String()); err != nil {
				log.Warnf("could not open %s in your default browser.", webUI)
			}
			return nil
		},
	}

}
