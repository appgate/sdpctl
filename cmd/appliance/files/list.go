package files

import (
	"context"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

func NewFilesListCmd(f *factory.Factory) *cobra.Command {
	opts := &FilesOptions{
		Config:    f.Config,
		Out:       f.IOOutWriter,
		Appliance: f.Appliance,
	}
	var listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   docs.FilesListDocs.Short,
		Long:    docs.FilesListDocs.Long,
		Example: docs.FilesListDocs.ExampleString(),
		Args: func(cmd *cobra.Command, args []string) error {
			var errs *multierror.Error
			var err error
			opts.OrderBy, err = cmd.Flags().GetStringSlice("order-by")
			if err != nil {
				errs = multierror.Append(errs, err)
			}
			opts.Descending, err = cmd.Flags().GetBool("descending")
			if err != nil {
				errs = multierror.Append(errs, err)
			}
			return errs.ErrorOrNil()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := opts.Appliance(f.Config)
			if err != nil {
				return err
			}

			ctx := context.Background()
			files, err := a.ListFiles(ctx, opts.OrderBy, opts.Descending)
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
