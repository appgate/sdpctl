package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/spf13/cobra"
)

type upgradeStatusOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug     bool
	json      bool
}

// NewUpgradeStatusCmd return a new upgrade status command
func NewUpgradeStatusCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeStatusOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var upgradeStatusCmd = &cobra.Command{
		Use:   "status",
		Short: `Display the upgrade status of Appgate SDP Appliances`,
		Long: `Display the upgrade status of Appgate SDP Appliances in either table or json format.
Upgrade statuses:
- idle:         No upgrade is initiated
- started:      Upgrade process has started
- downloading:  Appliance is downloading the upgrade image
- verifying:    Upgrade image download is completed and the image is being verified
- ready:        Image is verified and ready to be applied
- installing:   Appliance is installing the upgrade image
- success:      Upgrade successful
- failed:       Upgrade failed for some reason during the process`,
		Example: `# View in table format
$ appgatectl appliance upgrade status

# View in JSON format
$ appgatectl appliance upgrade status --json

# Filter appliances
$ appgatectl appliance upgrade status --filter=name=controller`,
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeStatusRun(c, args, &opts)
		},
	}

	flags := upgradeStatusCmd.Flags()
	flags.BoolVar(&opts.json, "json", false, "Display in JSON format")

	return upgradeStatusCmd
}

func upgradeStatusRun(cmd *cobra.Command, args []string, opts *upgradeStatusOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	filter := util.ParseFilteringFlags(cmd.Flags())
	allAppliances, err := a.List(ctx, filter)
	if err != nil {
		return err
	}
	initialStats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	appliances, offline, _ := appliancepkg.FilterAvailable(allAppliances, initialStats.GetData())

	type ApplianceStatus struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Status        string `json:"status,omitempty"`
		UpgradeStatus string `json:"upgrade_status,omitempty"`
		Details       string `json:"details,omitempty"`
	}
	statuses := make([]ApplianceStatus, 0, len(appliances))
	for _, appliance := range allAppliances {
		id := appliance.GetId()
		mode := "online"
		for _, o := range offline {
			if o.GetId() == id {
				mode = "offline"
			}
		}
		row := ApplianceStatus{
			ID:     id,
			Name:   appliance.GetName(),
			Status: mode,
		}
		if mode == "online" && appliance.GetActivated() {
			status, err := a.UpgradeStatus(ctx, id)
			if err != nil {
				return err
			}
			row.UpgradeStatus = status.GetStatus()
			row.Details = status.GetDetails()
		}

		statuses = append(statuses, row)
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
	fmt.Fprintln(w, "ID\tName\tStatus\tUpgrade Status\tDetails")
	for _, s := range statuses {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.ID, s.Name, s.Status, s.UpgradeStatus, s.Details)
	}
	w.Flush()
	return nil
}
