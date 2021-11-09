package appliance

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

	util "github.com/appgate/appgatectl/internal"
	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/cmd/factory"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"

	log "github.com/sirupsen/logrus"
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
	if appliance.IsOnAppliance() {
		return appliance.ErrExecutedOnAppliance
	}
	if opts.image == "" {
		return errors.New("Image is mandatory")
	}

	if ok, err := appliance.FileExists(opts.image); err != nil || !ok {
		return fmt.Errorf("Image file not found %q", opts.image)
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
	targetVersion, err := appliance.GuessVersion(filename)
	if err != nil {
		log.Debugf("Could not guess target version based on the image file name %q", filename)
	}
	appliances, err := appliance.GetAllAppliances(ctx, client, token)
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
	_, err = appliance.GetFileStatus(ctx, client, token, filename)
	if err != nil {
		// if we dont get 404, return err
		if !errors.Is(err, appliance.ErrFileNotFound) {
			return err
		}
	}
	log.Infof("Uploading %q to controller", f.Name())
	if err := appliance.UploadFile(ctx, client, token, f); err != nil {
		return err
	}
	log.Infof("Uploaded %q to controller", f.Name())
	remoteFile, err := appliance.GetFileStatus(ctx, client, token, filename)
	if err != nil {
		// if we dont get 404, return err
		if !errors.Is(err, appliance.ErrFileNotFound) {
			return err
		}
	}
	log.Infof("Remote file %s is %s", remoteFile.GetName(), remoteFile.GetStatus())
	// Step 2
	remoteFilePath := fmt.Sprintf("controller://%s:%d/%s", primaryController.GetHostname(), 8443, filename)
	for _, a := range appliances {
		if err := appliance.PrepareFileOn(ctx, client, token, remoteFilePath, a.GetId()); err != nil {
			log.Warnf("Failed to prepare %s %s", a.GetName(), err)
		}
	}
	// Step 3
	if err := appliance.DeleteFile(ctx, client, token, filename); err != nil {
		log.Warnf("Failed to delete %s from controller %s", filename, err)
	}
	fmt.Fprintf(opts.Out, "\n Ok fin. \n")
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

func activeApplianceFunctions(appliances []openapi.Appliance) map[string]bool {
	functions := make(map[string]bool, 0)
	for _, a := range appliances {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			functions["controller"] = true
		}
		if v, ok := a.GetGatewayOk(); ok && v.GetEnabled() {
			functions["gateway"] = true
		}
		if v, ok := a.GetPortalOk(); ok && v.GetEnabled() {
			functions["portal"] = true
		}
		if v, ok := a.GetConnectorOk(); ok && v.GetEnabled() {
			functions["connector"] = true
		}
		if v, ok := a.GetLogServerOk(); ok && v.GetEnabled() {
			functions["log_server"] = true
		}
	}
	return functions
}

func applianceGroupDescription(appliances []openapi.Appliance) string {
	functions := activeApplianceFunctions(appliances)
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
