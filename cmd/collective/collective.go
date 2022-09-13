package collective

import (
	"io"

	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

type commandOpts struct {
	Out io.Writer
}

// NewCollectiveCmd return a new collective subcommand
func NewCollectiveCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use: "collective",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		TraverseChildren: true,
		Short:            "",
		Long:             "",
	}
	opts := &commandOpts{
		Out: f.IOOutWriter,
	}

	return cmd
}
