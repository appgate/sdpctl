package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type upgradeStatusOptions struct {
	Config        *configuration.Config
	Out           io.Writer
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug         bool
	json          bool
	defaultFilter map[string]map[string]string
}

// NewUpgradeStatusCmd return a new upgrade status command
func NewUpgradeStatusCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeStatusOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		defaultFilter: map[string]map[string]string{
			"include": {},
			"exclude": {
				"active": "false",
			},
		},
	}
	var upgradeStatusCmd = &cobra.Command{
		Use:     "status",
		Short:   docs.ApplianceUpgradeStatusDoc.Short,
		Long:    docs.ApplianceUpgradeStatusDoc.Long,
		Example: docs.ApplianceUpgradeStatusDoc.ExampleString(),
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
	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), opts.defaultFilter)
	allAppliances, err := a.List(ctx, filter, orderBy, descending)
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

	w := util.NewPrinter(opts.Out, 4)
	w.AddHeader("ID", "Name", "Status", "Upgrade Status", "Details")
	for _, s := range statuses {
		w.AddLine(s.ID, s.Name, s.Status, s.UpgradeStatus, s.Details)
	}
	w.Print()
	return nil
}
