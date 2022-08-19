package files

import (
	"context"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewFilesListCmd(f *factory.Factory) *cobra.Command {
	opts := &FilesOptions{
		Config: f.Config,
		Out:    f.IOOutWriter,
		API:    f.Files,
	}
	var listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   docs.FilesListDocs.Short,
		Long:    docs.FilesListDocs.Long,
		Example: docs.FilesListDocs.ExampleString(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			f, err := opts.API(opts.Config)
			if err != nil {
				return err
			}

			files, err := f.List(ctx)
			if err != nil {
				return err
			}

			if opts.JSON {
				return util.PrintJSON(opts.Out, files)
			}

			p := util.NewPrinter(opts.Out, 4)
			p.AddHeader("Name", "Status", "Created", "Modified", "Failure Reason")
			for _, file := range files {
				p.AddLine(file.GetName(), file.GetStatus(), file.GetCreationTime(), file.GetLastModifiedTime(), file.GetFailureReason())
			}
			p.Print()

			return nil
		},
	}

	listCmd.Flags().BoolVar(&opts.JSON, "json", false, "output in json format")

	return listCmd
}
