package profile

import (
	"fmt"
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

// NewListCmd return a new profile list command
func NewListCmd(opts *commandOpts) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   docs.ProfileListDoc.Short,
		Long:    docs.ProfileListDoc.Long,
		RunE: func(c *cobra.Command, args []string) error {
			return listRun(c, args, opts)
		},
	}
}

func listRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !profiles.FileExists() {
		fmt.Fprintln(opts.Out, "no profiles added")
		return nil
	}

	p, err := profiles.Read()
	if err != nil {
		return err
	}
	currentProfile, err := p.CurrentProfile()
	if err != nil {
		fmt.Fprintf(opts.Out, "%s\n", err.Error())
	}
	if currentProfile != nil {
		currentConfig, err := readConfig(filepath.Join(currentProfile.Directory, "config.json"))
		if err != nil {
			fmt.Fprintf(opts.Out, "Current profile %s is not configure, run 'sdpctl configure'\n", currentProfile.Name)
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
	fmt.Fprintf(opts.Out, "\nAvailable profiles\n")
	printer := util.NewPrinter(opts.Out, 4)
	printer.AddHeader("Name", "Config directory")
	for _, profile := range p.List {
		printer.AddLine(profile.Name, profile.Directory)
	}
	printer.Print()
	return nil
}
