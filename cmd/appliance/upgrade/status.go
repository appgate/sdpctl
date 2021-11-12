package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
)

type upgradeStatusOptions struct {
	Config     *configuration.Config
	Out        io.Writer
	Appliance  func(c *configuration.Config) (*appliance.Appliance, error)
	Token      string
	Timeout    int
	url        string
	provider   string
	debug      bool
	insecure   bool
	apiversion int
	cacert     string
	json       bool
}

// NewUpgradeStatusCmd return a new upgrade status command
func NewUpgradeStatusCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeStatusOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		Timeout:   10,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var upgradeStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "upgrade status",
		Long:  `TODO`,
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeStatusRun(c, args, &opts)
		},
	}

	upgradeStatusCmd.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	upgradeStatusCmd.PersistentFlags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	upgradeStatusCmd.PersistentFlags().StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	upgradeStatusCmd.PersistentFlags().IntVarP(&opts.apiversion, "apiversion", "", f.Config.Version, "peer API version")
	upgradeStatusCmd.PersistentFlags().StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	upgradeStatusCmd.PersistentFlags().StringVarP(&opts.cacert, "cacert", "", "", "Path to the controller's CA cert file in PEM or DER format")

	return upgradeStatusCmd
}

func upgradeStatusRun(cmd *cobra.Command, args []string, opts *upgradeStatusOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	appliances, err := a.GetAll(ctx)
	if err != nil {
		return err
	}
	type ApplianceStatus struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Status  string `json:"status,omitempty"`
		Details string `json:"details,omitempty"`
	}
	statuses := make([]ApplianceStatus, 0, len(appliances))
	for _, appliance := range appliances {
		id := appliance.GetId()
		status, err := a.UpgradeStatus(ctx, id)
		if err != nil {
			return err
		}
		statuses = append(statuses, ApplianceStatus{
			ID:      id,
			Name:    appliance.GetName(),
			Status:  status.GetStatus(),
			Details: status.GetDetails(),
		})
	}
	if opts.json {
		jsonStatus, err := json.MarshalIndent(&statuses, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", string(jsonStatus))
		return nil
	}

	w := tabwriter.NewWriter(opts.Out, 4, 4, 8, ' ', tabwriter.DiscardEmptyColumns)
	fmt.Fprintln(w, "ID\tName\tStatus\tDetails")
	for _, s := range statuses {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Name, s.Status, s.Details)
	}
	w.Flush()
	return nil
}
