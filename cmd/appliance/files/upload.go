package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	apipkg "github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
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
	filePaths map[string]string
}

func NewFilesUploadCmd(f *factory.Factory) *cobra.Command {
	opts := uploadOptions{
		config:    f.Config,
		out:       f.IOOutWriter,
		filePaths: map[string]string{},
	}
	uploadCMD := &cobra.Command{
		Use:     "upload",
		Aliases: []string{"up"},
		Short:   docs.FilesUploadDocs.Short,
		Long:    docs.FilesUploadDocs.Long,
		Example: docs.FilesUploadDocs.ExampleString(),
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var errs *multierror.Error
			var err error

			if opts.ciMode, err = cmd.Flags().GetBool("ci-mode"); err != nil {
				return err
			}

			for _, arg := range args {
				pathSlice := strings.Split(arg, "=")
				path := filesystem.AbsolutePath(pathSlice[0])
				var rename string
				if len(pathSlice) > 1 {
					rename = filepath.Base(pathSlice[1])
				}
				ok, err := util.FileExists(path)
				if err != nil {
					errs = multierror.Append(errs, err)
					continue
				}
				if !ok {
					errs = multierror.Append(errs, fmt.Errorf("file does not exist: %s", path))
					continue
				}
				opts.filePaths[path] = rename
			}

			return errs.ErrorOrNil()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			api, err := f.Appliance(f.Config)
			if err != nil {
				return err
			}

			ctx := util.BaseAuthContext(api.Token)
			filesAPI := files.FileManager{API: api}
			if !opts.ciMode {
				filesAPI.Progress = tui.New(ctx, f.SpinnerOut)
			}

			tempPaths := map[string]string{}
			for path, rename := range opts.filePaths {
				name := filepath.Base(path)
				if len(rename) > 0 {
					name = rename
				}
				_, err := api.FileStatus(ctx, name)
				if err != nil {
					if errors.Is(err, apipkg.ErrFileNotFound) {
						tempPaths[path] = rename
						continue
					}
					return err
				}
				// no error means file already exists in the repository
				if f.CanPrompt() {
					overwrite, err := prompt.PromptConfirm(fmt.Sprintf("%s already exists. Overwrite?", name), false)
					if err != nil {
						return err
					}
					if !overwrite {
						continue
					}
				}
				tempPaths[path] = rename
			}

			if len(tempPaths) <= 0 {
				fmt.Fprintln(f.IOOutWriter, "No files to upload.")
				return nil
			}

			// Delete all files that are being uploaded first
			for path := range tempPaths {
				name := filepath.Base(path)
				api.DeleteFile(ctx, name)
			}

			errChan := make(chan error)
			var wg sync.WaitGroup
			var errs *multierror.Error
			fmt.Fprintf(f.IOOutWriter, "Uploading %d file(s):\n\n", len(tempPaths))
			for f, rename := range tempPaths {
				file, err := os.Open(f)
				if err != nil {
					errs = multierror.Append(errs, err)
					continue
				}
				wg.Add(1)
				go func(ctx context.Context, wg *sync.WaitGroup, err chan<- error, f *os.File, filesAPI *files.FileManager, rename string) {
					defer func() {
						wg.Done()
						f.Close()
					}()
					err <- filesAPI.Upload(ctx, files.QueueItem{File: f, RemoteName: rename})
				}(ctx, &wg, errChan, file, &filesAPI, rename)
			}

			go func(wg *sync.WaitGroup, errChan chan<- error) {
				wg.Wait()
				close(errChan)
			}(&wg, errChan)

			for result := range errChan {
				if result != nil {
					errs = multierror.Append(errs, result)
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
