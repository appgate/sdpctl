package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	util "github.com/appgate/appgatectl/internal"
	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/cmd/factory"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type prepareUpgradeOptions struct {
	Config     *config.Config
	Out        io.Writer
	Appliance  func(c *config.Config) (*appliance.Appliance, error)
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
		Appliance: f.Appliance,
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
	if appliance.IsOnAppliance() {
		return appliance.ErrExecutedOnAppliance
	}
	if opts.image == "" {
		return errors.New("Image is mandatory")
	}

	if ok, err := appliance.FileExists(opts.image); err != nil || !ok {
		return fmt.Errorf("Image file not found %q", opts.image)
	}

	a, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}
	f, err := os.Open(opts.image)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Minute))
	defer cancel()

	filename := filepath.Base(f.Name())
	targetVersion, err := appliance.GuessVersion(filename)
	if err != nil {
		log.Debugf("Could not guess target version based on the image file name %q", filename)
	}
	appliances, err := a.GetAll(ctx)
	if err != nil {
		return err
	}
	peerAppliances := appliancesWithAdminOnPeerInterface(appliances)
	if len(peerAppliances) > 0 {
		msg, err := showPeerInterfaceWarningMessage(peerAppliances)
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", msg)
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}
	u, err := url.Parse(opts.url)
	if err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}

	primaryController, err := appliance.FindPrimaryController(appliances, host)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", fmt.Sprintf(appliance.BackupInstructions, primaryController.Name, appliance.HelpManualURL))
	if err := prompt.AskConfirmation("Have you completed the Controller backup or snapshot?"); err != nil {
		return err
	}
	log.Infof("Primary controller is: %q", primaryController.Name)
	if targetVersion != nil {
		log.Infof("Appliances will be prepared for upgrade to version: %s", targetVersion.String())
	}
	msg, err := showPrepareUpgradeMessage(f.Name(), appliances)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", msg)
	if err := prompt.AskConfirmation(); err != nil {
		return err
	}

	// Step 1
	_, err = a.FileStatus(ctx, filename)
	if err != nil {
		// if we dont get 404, return err
		if !errors.Is(err, appliance.ErrFileNotFound) {
			return err
		}
	}
	log.Infof("Uploading %q to controller", f.Name())
	if err := a.UploadFile(ctx, f); err != nil {
		return err
	}
	log.Infof("Uploaded %q to controller", f.Name())
	remoteFile, err := a.FileStatus(ctx, filename)
	if err != nil {
		// if we dont get 404, return err
		if !errors.Is(err, appliance.ErrFileNotFound) {
			return err
		}
	}
	log.Infof("Remote file %s is %s", remoteFile.GetName(), remoteFile.GetStatus())
	// Step 2
	prepare := func(ctx context.Context, primaryController openapi.Appliance, appliances []openapi.Appliance) error {
		remoteFilePath := fmt.Sprintf("controller://%s:%d/%s", primaryController.GetHostname(), 8443, filename)
		g, ctx := errgroup.WithContext(ctx)
		for _, appliance := range appliances {
			i := appliance // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() error {
				log.Infof("Preparing upgrade for %s", i.GetName())
				if err := a.PrepareFileOn(ctx, remoteFilePath, i.GetId()); err != nil {
					return fmt.Errorf("Failed to prepare %s %s", i.GetName(), err)
				}
				status, err := a.UpgradeStatus(ctx, i.GetId())
				if err != nil {
					return fmt.Errorf("Failed to get status after prepare %s %s", i.GetName(), err)
				}
				log.Infof("%s status is %s - %s", i.GetName(), status.GetStatus(), status.GetDetails())
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
		return nil
	}
	if err := prepare(ctx, *primaryController, appliances); err != nil {
		return err
	}

	// Step 3
	log.Infof("3. Delete upgrade image %s from Controller", filename)
	if err := a.DeleteFile(ctx, filename); err != nil {
		log.Warnf("Failed to delete %s from controller %s", filename, err)
	}
	log.Infof("File %s deleted from Controller", filename)

	return nil
}

// appliancesWithAdminOnPeerInterface List all appliances still using the peer interface for the admin API, this is now deprecated.
func appliancesWithAdminOnPeerInterface(appliances []openapi.Appliance) []openapi.Appliance {
	peer := make([]openapi.Appliance, 0)
	for _, a := range appliances {
		if _, ok := a.GetAdminInterfaceOk(); !ok {
			peer = append(peer, a)
		}
	}
	return peer
}

func appliancePeerPorts(appliances []openapi.Appliance) string {
	ports := make([]int, 0)
	for _, a := range appliances {
		if v, ok := a.GetPeerInterfaceOk(); ok {
			if v, ok := v.GetHttpsPortOk(); ok && *v > 0 {
				ports = util.AppendIfMissing(ports, int(*v))
			}
		}
	}
	return strings.Trim(strings.Replace(fmt.Sprint(ports), " ", ",", -1), "[]")
}

func applianceGroupDescription(appliances []openapi.Appliance) string {
	functions := appliance.ActiveFunctions(appliances)
	var funcs []string
	for k, value := range functions {
		if _, ok := functions[k]; ok && value {
			funcs = append(funcs, k)
		}
	}
	return strings.Join(funcs, ", ")
}

const prepareUpgradeMessage = `
1. Upload upgrade image {{.Filepath}} to Controller
2. Prepare upgrade on the following appliances:
{{range .Appliances }}
  - {{.Name -}}
{{end}}

3. Delete upgrade image from Controller
`

func showPrepareUpgradeMessage(f string, appliance []openapi.Appliance) (string, error) {
	type stub struct {
		Filepath   string
		Appliances []openapi.Appliance
	}
	data := stub{
		Filepath:   f,
		Appliances: appliance,
	}
	t := template.Must(template.New("").Parse(prepareUpgradeMessage))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}

const applianceUsingPeerWarning = `
Version 5.4 and later are designed to operate with the admin port (default 8443)
separate from the deprecated peer port (set to {{.CurrentPort}}).
It is recommended to switch to port 8443 before continuing
The following {{.Functions}} {{.Noun}} still configured without the Admin/API TLS Connection:
{{range .Appliances}}
  - {{.Name -}}
{{end}}
`

func showPeerInterfaceWarningMessage(peerAppliances []openapi.Appliance) (string, error) {
	type stub struct {
		CurrentPort string
		Functions   string
		Noun        string
		Appliances  []openapi.Appliance
	}
	noun := "are"
	if len(peerAppliances) == 1 {
		noun = "is"
	}
	data := stub{
		CurrentPort: appliancePeerPorts(peerAppliances),
		Functions:   applianceGroupDescription(peerAppliances),
		Noun:        noun,
		Appliances:  peerAppliances,
	}
	t := template.Must(template.New("peer").Parse(applianceUsingPeerWarning))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
