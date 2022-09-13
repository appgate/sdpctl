package collective

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

// NewListCmd return a new collective list command
func NewListCmd(opts *commandOpts) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "",
		Long:    "",
		RunE: func(c *cobra.Command, args []string) error {
			return listRun(c, args, opts)
		},
	}
}

func listRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !configuration.ProfileFileExists() {
		fmt.Fprintln(opts.Out, "no profiles added")
		return nil
	}

	content, err := os.ReadFile(configuration.ProfileFilePath())
	if err != nil {
		return fmt.Errorf("Can't read profiles: %s %s\n", configuration.ProfileFilePath(), err)
	}

	var profiles configuration.Profiles
	if err := json.Unmarshal(content, &profiles); err != nil {
		return fmt.Errorf("%s file is corrupt: %s \n", configuration.ProfileFilePath(), err)
	}

	currentProfile, err := profiles.CurrentProfile()
	if err != nil {
		fmt.Fprintf(opts.Out, "%s\n", err.Error())
	}
	if currentProfile != nil {
		currentConfig, err := currentProfile.CurrentConfig()
		if err != nil {
			fmt.Fprintf(opts.Out, "%s, run 'sdpctl configure'", err.Error())
		}

		if currentConfig != nil {
			h, err := currentConfig.GetHost()
			if err != nil {
				fmt.Fprintf(opts.Out, "Current profile %s is not configure, run 'sdpctl configure'\n", currentProfile.Name)
			} else {
				fmt.Fprintf(opts.Out, "Current profile is %s (%s) primary controller %s\n", currentProfile.Name, currentProfile.Directory, h)
			}
		}
	}

	fmt.Fprintf(opts.Out, "\nAvailable collective profiles\n")
	p := util.NewPrinter(opts.Out, 4)
	p.AddHeader("Name", "Config directory")
	for _, profile := range profiles.List {
		p.AddLine(profile.Name, profile.Directory)
	}
	p.Print()
	return nil
}
