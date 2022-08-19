package files

import (
	"io"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/files"
	"github.com/spf13/cobra"
)

type FilesOptions struct {
	Config *configuration.Config
	Out    io.Writer
	API    func(c *configuration.Config) (*files.FilesAPI, error)
	JSON   bool
}

func NewFilesCmd(f *factory.Factory) *cobra.Command {
	var filesCmd = &cobra.Command{
		Use:     "files",
		Short:   docs.FilesDocs.Short,
		Long:    docs.FilesDocs.Long,
		Example: docs.FilesDocs.ExampleString(),
	}

	filesCmd.AddCommand(NewFilesListCmd(f))

	return filesCmd
}
