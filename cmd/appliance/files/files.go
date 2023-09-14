package files

import (
	"io"

	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

type FilesOptions struct {
	Config     *configuration.Config
	Appliance  func(c *configuration.Config) (*appliance.Appliance, error)
	Out        io.Writer
	JSON       bool
	OrderBy    []string
	Descending bool
}

func NewFilesCmd(f *factory.Factory) *cobra.Command {
	var filesCmd = &cobra.Command{
		Use:     "files",
		Short:   docs.FilesDocs.Short,
		Long:    docs.FilesDocs.Long,
		Example: docs.FilesDocs.ExampleString(),
	}

	filesCmd.AddCommand(NewFilesListCmd(f))
	filesCmd.AddCommand(NewFilesDeleteCmd(f))
	filesCmd.AddCommand(NewFilesUploadCmd(f))

	return filesCmd
}
