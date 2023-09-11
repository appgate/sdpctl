package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	apipkg "github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/files"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

type uploadOptions struct {
	config    *configuration.Config
	out       io.Writer
	ciMode    bool
	filePaths []string
}

func NewFilesUploadCmd(f *factory.Factory) *cobra.Command {
	opts := uploadOptions{
		config: f.Config,
		out:    f.IOOutWriter,
	}
	uploadCMD := &cobra.Command{
		Use:     "upload",
		Aliases: []string{"up"},
		Short:   "",
		Long:    "",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var errs *multierror.Error
			var err error

			if opts.ciMode, err = cmd.Flags().GetBool("ci-mode"); err != nil {
				return err
			}

			for _, arg := range args {
				path := filesystem.AbsolutePath(arg)
				ok, err := util.FileExists(path)
				if err != nil {
					errs = multierror.Append(errs, err)
					continue
				}
				if !ok {
					errs = multierror.Append(errs, fmt.Errorf("file does not exist: %s", path))
					continue
				}
				opts.filePaths = append(opts.filePaths, path)
			}

			return errs.ErrorOrNil()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			api, err := f.Appliance(f.Config)
			if err != nil {
				return err
			}

			ctx := context.Background()
			filesAPI := files.FilesAPI{API: api}
			if !opts.ciMode {
				filesAPI.Progress = tui.New(ctx, f.SpinnerOut)
			}

			tempPaths := []string{}
			for _, path := range opts.filePaths {
				name := filepath.Base(path)
				_, err := api.FileStatus(ctx, name)
				if err != nil {
					if errors.Is(err, apipkg.ErrFileNotFound) {
						tempPaths = append(tempPaths, path)
						continue
					}
					return err
				}
				// no error means file already exists in the repository
				if f.CanPrompt() {
					p := &survey.Confirm{
						Message: fmt.Sprintf("%s already exists. Overwrite?", name),
					}
					var overwrite bool
					if err := prompt.SurveyAskOne(p, &overwrite); err != nil {
						return err
					}
					if !overwrite {
						continue
					}
				}
				tempPaths = append(tempPaths, path)
			}

			if len(tempPaths) <= 0 {
				fmt.Fprintln(f.IOOutWriter, "No files to upload.")
				return nil
			}

			// Delete all files that are being uploaded first
			for _, path := range tempPaths {
				name := filepath.Base(path)
				api.DeleteFile(ctx, name)
			}

			errChan := make(chan error)
			var wg sync.WaitGroup
			var errs *multierror.Error
			fmt.Fprintf(f.IOOutWriter, "Uploading %d file(s):\n\n", len(tempPaths))
			for _, f := range tempPaths {
				file, err := os.Open(f)
				if err != nil {
					errs = multierror.Append(errs, err)
					continue
				}
				wg.Add(1)
				go func(ctx context.Context, wg *sync.WaitGroup, err chan<- error, f *os.File, filesAPI *files.FilesAPI) {
					defer func() {
						wg.Done()
						f.Close()
					}()
					err <- filesAPI.Upload(ctx, f)
				}(ctx, &wg, errChan, file, &filesAPI)
			}

			go func(wg *sync.WaitGroup, errChan chan<- error) {
				wg.Wait()
				close(errChan)
			}(&wg, errChan)

			for result := range errChan {
				if result != nil {
					multierror.Append(errs, result)
				}
			}

			if filesAPI.Progress != nil {
				filesAPI.Progress.Wait()
			}

			fmt.Fprintln(f.IOOutWriter, "\nFile upload complete!")

			return errs.ErrorOrNil()
		},
	}

	return uploadCMD
}
