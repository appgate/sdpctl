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
		Config: f.Config,
		Out:    f.IOOutWriter,
		API:    f.Files,
	}

	deleteCmd := &cobra.Command{
		Use:       "delete",
		Aliases:   []string{"remove", "rm"},
		Short:     docs.FilesDeleteDocs.Short,
		Long:      docs.FilesDeleteDocs.Long,
		Example:   docs.FilesDeleteDocs.ExampleString(),
		ValidArgs: []string{"filename"},
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := opts.API(f.Config)
			if err != nil {
				return err
			}

			ctx := context.Background()

			if len(args) == 1 {
				if err := api.Delete(ctx, args[0]); err != nil {
					return err
				}
				fmt.Fprintf(opts.Out, "%s: deleted\n", args[0])
				return nil
			}

			allFlag, err := cmd.Flags().GetBool("all")
			if err != nil {
				return err
			}

			fileList, err := api.List(ctx)
			if err != nil {
				return err
			}

			if allFlag {
				for _, file := range fileList {
					if err := api.Delete(ctx, file.GetName()); err != nil {
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

				for _, s := range selected {
					if err := api.Delete(ctx, s); err != nil {
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
