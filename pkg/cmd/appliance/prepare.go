package appliance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	log.Infof("Primary controller is: %q", primaryController.Name)

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

const applianceUsingPeerWarning = `
Version 5.4 and later are designed to operate with the admin port (default 8443)
separate from the deprecated peer port (set to {{.CurrentPort}}).
It is recommended to switch to port 8443 before continuing
The following {{.Functions}} {{.Noun}} still configured without the Admin/API TLS Connection:
{{range .Appliances}}
  - {{.Name}}
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
