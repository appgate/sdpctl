package files

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := opts.Appliance(f.Config)
			if err != nil {
				return err
			}

			ctx := context.Background()

			if len(args) == 1 {
				if err := a.DeleteFile(ctx, args[0]); err != nil {
					return err
				}
				fmt.Fprintf(opts.Out, "%s: deleted\n", args[0])
				return nil
			}

			allFlag, err := cmd.Flags().GetBool("all")
			if err != nil {
				return err
			}

			fileList, err := a.ListFiles(ctx)
			if err != nil {
				return err
			}

			if allFlag {
				for _, file := range fileList {
					if err := a.DeleteFile(ctx, file.GetName()); err != nil {
						return err
					}
					fmt.Fprintf(opts.Out, "%s: deleted\n", file.GetName())
				}
				return nil
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
				qs := &survey.MultiSelect{
					PageSize: len(fileNameList),
					Message:  "select files to delete:",
					Options:  fileNameList,
				}

				selected := []string{}
				if err := prompt.SurveyAskOne(qs, &selected); err != nil {
					return err
				}

				if len(selected) <= 0 {
					return errors.New("No files were selected for deletion")
				}
				for _, s := range selected {
					if err := a.DeleteFile(ctx, s); err != nil {
						return err
					}
					fmt.Fprintf(opts.Out, "%s: deleted\n", s)
				}

				return nil
			}

			return errors.New("No files were deleted")
		},
	}

	deleteCmd.Flags().Bool("all", false, "delete all files from repository")

	return deleteCmd
}
