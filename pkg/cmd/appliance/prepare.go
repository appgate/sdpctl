package appliance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/cmd/factory"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/spf13/cobra"
)

type prepareUpgradeOptions struct {
	Config     *config.Config
	Out        io.Writer
	APIClient  func(Config *config.Config) (*openapi.APIClient, error)
	Token      string
	Timeout    int
	url        string
	provider   string
	debug      bool
	insecure   bool
	apiversion int
	cacert     string
	image      string
}

// NewPrepareUpgradeCmd return a new prepare upgrade command
func NewPrepareUpgradeCmd(f *factory.Factory) *cobra.Command {
	opts := &prepareUpgradeOptions{
		Config:    f.Config,
		APIClient: f.APIClient,
		Timeout:   10,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var prepareCmd = &cobra.Command{
		Use:   "prepare",
		Short: "prepare upgrade",
		Long:  `TODO`,
		RunE: func(c *cobra.Command, args []string) error {
			return prepareRun(c, args, opts)
		},
	}

	prepareCmd.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	prepareCmd.PersistentFlags().StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	prepareCmd.PersistentFlags().IntVarP(&opts.apiversion, "apiversion", "", f.Config.Version, "peer API version")
	prepareCmd.PersistentFlags().StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	prepareCmd.PersistentFlags().StringVarP(&opts.cacert, "cacert", "", "", "Path to the controller's CA cert file in PEM or DER format")
	prepareCmd.PersistentFlags().StringVarP(&opts.image, "image", "", "", "image path")

	return prepareCmd
}

func prepareRun(cmd *cobra.Command, args []string, opts *prepareUpgradeOptions) error {
	fmt.Println("upgrade prepare placeholder")
	if opts.image == "" {
		return fmt.Errorf("Image is mandatory")
	}
	client, err := opts.APIClient(opts.Config)
	if err != nil {
		return err
	}
	f, err := os.Open(opts.image)
	if err != nil {
		return err
	}
	ctx := context.Background()
	token := opts.Config.GetBearTokenHeaderValue()
	filename := filepath.Base(f.Name())
	_, err = appliance.GetFileStatus(ctx, client, token, filename)
	if err != nil {
		// if we dont get 404, return err
		if !errors.Is(err, appliance.ErrFileNotFound) {
			return err
		}
	}

	if err := appliance.UploadFile(ctx, client, token, f); err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n Ok continue \n")
	return nil
}
