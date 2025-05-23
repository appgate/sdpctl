package functions

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logServerImages []string = []string{
		"cz-opensearch",
		"cz-opensearchdashboards",
	}
	funcImages map[string][]string = map[string][]string{
		appliancepkg.FunctionLogServer: logServerImages,
	}
)

type bundleOptions struct {
	registry    *url.URL
	destination string
	version     string
	ciMode      bool
	out         io.Writer
}

func NewApplianceFunctionsDownloadCmd(f *factory.Factory) *cobra.Command {
	opts := bundleOptions{
		out: f.IOOutWriter,
	}
	cmd := &cobra.Command{
		Use:     "download [function...]",
		Aliases: []string{"dl", "bundle"},
		Annotations: map[string]string{
			configuration.SkipAuthCheck: "true",
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return ValidFuncs, cobra.ShellCompDirectiveNoFileComp
		},
		Short:   docs.ApplianceFunctionsDownloadDocs.Short,
		Long:    docs.ApplianceFunctionsDownloadDocs.Long,
		Example: docs.ApplianceFunctionsDownloadDocs.ExampleString(),
		Args: cobra.MatchAll(cobra.MinimumNArgs(1), func(cmd *cobra.Command, args []string) error {
			registry, err := cmd.Flags().GetString("docker-registry")
			if err != nil {
				return err
			}
			if v, err := cmd.Flags().GetString("save-path"); err == nil && len(v) > 0 {
				fmt.Fprintln(opts.out, "WARNING: the 'save-path' flag is deprecated. Please use the 'destination' flag instead.")
				opts.destination = v
			}
			if opts.registry, err = f.DockerRegistry(registry); err != nil {
				return err
			}
			if tag, err := cmd.Flags().GetString("version"); err == nil && len(tag) > 0 {
				if v, err := appliancepkg.ParseVersionString(tag); err == nil {
					opts.version, err = util.DockerTagVersion(v)
					if err != nil {
						return err
					}
				}
			}
			if opts.ciMode, err = cmd.Flags().GetBool("ci-mode"); err != nil {
				return err
			}

			var errs *multierror.Error
			for _, a := range args {
				if util.InSlice(a, UnavailableFuncs) {
					errs = multierror.Append(errs, fmt.Errorf("Function not yet supported: '%s'", a))
					continue
				}
				if !util.InSlice(a, ValidFuncs) {
					errs = multierror.Append(errs, fmt.Errorf("Invalid function provided: '%s'", a))
				}
			}
			return errs.ErrorOrNil()
		}),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.destination = filesystem.AbsolutePath(opts.destination)
			if _, err := os.Stat(opts.destination); err != nil {
				createDir := true
				if f.CanPrompt() {
					createDir, err = prompt.PromptConfirm(fmt.Sprintf("Directory '%s' does not exist. Do you want to create it now?", opts.destination), true)
					if err != nil {
						return err
					}
				} else {
					fmt.Fprintf(opts.out, "Directory '%s' does not exist\n", opts.destination)
				}
				if createDir {
					fmt.Fprintf(opts.out, "creating directory '%s'\n", opts.destination)
					if err := os.MkdirAll(opts.destination, 0o700); err != nil {
						return err
					}
				}
			}

			if v, err := version.NewVersion(opts.version); err == nil {
				x, _ := version.NewVersion("6.2.0")
				if x.GreaterThan(v) {
					return fmt.Errorf("unsupported version: %s, only available for version 6.2 or higher", opts.version)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.HTTPClient()
			if err != nil {
				return err
			}
			var wg sync.WaitGroup
			errChan := make(chan error)
			ctx := context.Background()
			var p *tui.Progress
			if !opts.ciMode {
				p = tui.New(ctx, f.SpinnerOut)
			}
			for _, arg := range args {
				wg.Add(1)
				go func(ctx context.Context, wg *sync.WaitGroup, function string, opts bundleOptions, errs chan<- error, p *tui.Progress) {
					defer wg.Done()
					zipName := fmt.Sprintf("%s-%s.zip", strings.ToLower(function), opts.version)
					path := filepath.Join(opts.destination, zipName)
					endMsg := fmt.Sprintf("bundle ready. saved as '%s'", path)
					var t *tui.Tracker
					if p != nil {
						t = p.AddTracker(fmt.Sprintf("%s %s", function, opts.version), "preparing", endMsg)
						go t.Watch([]string{endMsg}, []string{"failed"})
						defer t.Update(endMsg)
					}

					images := make(map[string]string, len(funcImages[function]))
					for _, f := range funcImages[function] {
						images[f] = opts.version
					}
					file, err := appliancepkg.DownloadDockerBundles(ctx, p, client, path, opts.registry, images, false)
					if err != nil {
						errs <- err
						os.Remove(file.Name())
						return
					}
					logrus.WithField("file", file.Name()).Info("bundle ready")
				}(ctx, &wg, arg, opts, errChan, p)
			}

			var errs *multierror.Error
			go func(errChan <-chan error) {
				for e := range errChan {
					errs = multierror.Append(errs, e)
				}
			}(errChan)

			wg.Wait()
			if p != nil {
				p.Wait()
			}
			return errs.ErrorOrNil()
		},
	}

	flags := cmd.Flags()
	flags.String("docker-registry", "", "docker registry for downloading image bundles")
	flags.StringVar(&opts.destination, "destination", filesystem.DownloadDir(), "path to a directory where the container bundle should be saved. The command will create a directory if it doesn't already exist")
	flags.String("save-path", "", "[DEPRECATED, use '--destination' instead] path to a directory where the container bundle should be saved. The command will create a directory if it doesn't already exist")
	flags.StringVar(&opts.version, "version", "", "Override the LogServer version that will be downloaded. Defaults to the same version as the primary controller.")
	cmd.MarkFlagRequired("version")

	return cmd
}
