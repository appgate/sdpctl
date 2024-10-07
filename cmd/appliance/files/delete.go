package files

import (
	"context"
	"errors"
	"fmt"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

func NewFilesDeleteCmd(f *factory.Factory) *cobra.Command {
	opts := &FilesOptions{
		Config:    f.Config,
		Out:       f.IOOutWriter,
		Appliance: f.Appliance,
	}

	deleteCmd := &cobra.Command{
		Use:       "delete",
		Aliases:   []string{"del", "remove", "rm"},
		Short:     docs.FilesDeleteDocs.Short,
		Long:      docs.FilesDeleteDocs.Long,
		Example:   docs.FilesDeleteDocs.ExampleString(),
		ValidArgs: []string{"filename"},
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

			var errs error
			if len(args) > 0 {
				for _, arg := range args {
					if err := a.DeleteFile(ctx, arg); err != nil {
						errs = multierror.Append(err, errs)
						continue
					}
					fmt.Fprintf(opts.Out, "%s: deleted\n", arg)
				}
				return errs
			}

			allFlag, err := cmd.Flags().GetBool("all")
			if err != nil {
				return err
			}

			fileList, err := a.ListFiles(ctx, opts.OrderBy, opts.Descending)
			if err != nil {
				return err
			}

			if allFlag {
				for _, file := range fileList {
					if err := a.DeleteFile(ctx, file.GetName()); err != nil {
						errs = multierror.Append(err, errs)
						continue
					}
					fmt.Fprintf(opts.Out, "%s: deleted\n", file.GetName())
				}
				return errs
			}

			noInteractive, err := cmd.Flags().GetBool("no-interactive")
			if err != nil {
				return err
			}
			if !noInteractive {
				fileNameList := []string{}
				for _, file := range fileList {
					fileNameList = append(fileNameList, file.GetName())
				}
				selected, err := tui.MultipleChoice("select files to delete: ", fileNameList)
				if err != nil {
					return err
				}

				if len(selected) <= 0 {
					return errors.New("No files were selected for deletion")
				}
				for _, s := range selected {
					if err := a.DeleteFile(ctx, fileNameList[s]); err != nil {
						errs = multierror.Append(err, errs)
						continue
					}
					fmt.Fprintf(opts.Out, "%s: deleted\n", fileNameList[s])
				}

				return errs
			}

			return errors.New("No files were deleted")
		},
	}

	deleteCmd.Flags().Bool("all", false, "delete all files from repository")

	return deleteCmd
}
