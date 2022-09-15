package collective

import (
	"fmt"

	"github.com/appgate/sdpctl/pkg/profiles"
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
	if !profiles.FileExists() {
		fmt.Fprintln(opts.Out, "no profiles added")
		return nil
	}

	p, err := profiles.Read()
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.Out, "\nAvailable collective profiles\n")
	printer := util.NewPrinter(opts.Out, 4)
	printer.AddHeader("Name", "Config directory")
	for _, profile := range p.List {
		printer.AddLine(profile.Name, profile.Directory)
	}
	printer.Print()
	return nil
}
