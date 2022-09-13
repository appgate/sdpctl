package collective

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

// NewDeleteCmd return a new collective delete command
func NewDeleteCmd(opts *commandOpts) *cobra.Command {

	return &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm"},
		Short:   "",
		Long:    "",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return deleteRun(c, args, opts)
		},
	}
}

func deleteRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !configuration.ProfileFileExists() {
		fmt.Fprintln(opts.Out, "no profiles added")
		return nil
	}
	key := args[0]

	profiles, err := configuration.ReadProfiles()
	if err != nil {
		return err
	}
	list := profiles.List

	found := false
	for index, profile := range list {
		if key == profile.Name {
			target := list[index]
			list = append(list[:index], list[index+1:]...)
			found = true
			if profiles.Current != nil && *profiles.Current == target.Directory {
				profiles.Current = nil
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("Did not find %q as a existing profile", key)
	}

	profiles.List = list

	foo := filepath.Join(configuration.ProfileDirecty, key)
	if ok, err := util.FileExists(foo); err == nil && ok {
		if err := os.RemoveAll(foo); err != nil {
			fmt.Fprintf(opts.Out, "could not remove profile directory %s %s", foo, err)
		}
	}

	file, err := json.MarshalIndent(profiles, "", " ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configuration.ProfileFilePath, file, 0644); err != nil {
		return err
	}

	return nil
}
