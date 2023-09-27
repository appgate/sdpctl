package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

// NewDeleteCmd return a new profile delete command
func NewDeleteCmd(opts *commandOpts) *cobra.Command {
	return &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm"},
		Short:   docs.ProfileDeleteDoc.Short,
		Long:    docs.ProfileDeleteDoc.Long,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return deleteRun(c, args, opts)
		},
		ValidArgsFunction: tabCompletion,
	}
}

func deleteRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !profiles.FileExists() {
		fmt.Fprintln(opts.Out, "no profiles added")
		return nil
	}
	key := args[0]

	p, err := profiles.Read()
	if err != nil {
		return err
	}
	list := p.List

	found := false
	for index, profile := range list {
		if key == profile.Name {
			target := list[index]
			list = append(list[:index], list[index+1:]...)
			found = true
			if p.Current != nil && *p.Current == target.Directory {
				p.Current = nil
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("Did not find %q as a existing profile", key)
	}

	p.List = list

	cfgDir, logDir := profiles.Directories()
	profileDir := filepath.Join(cfgDir, key)
	if ok, err := util.FileExists(profileDir); err == nil && ok {
		if err := os.RemoveAll(profileDir); err != nil {
			fmt.Fprintf(opts.Out, "failed to remove profile directory %s %s", profileDir, err)
		}
	}
	logsDir := filepath.Join(logDir, key)
	if ok, err := util.FileExists(logsDir); err == nil && ok {
		if err := os.RemoveAll(profileDir); err != nil {
			fmt.Fprintf(opts.Out, "failed to remove profile log directory %s %s", logsDir, err)
		}
	}

	if err := profiles.Write(p); err != nil {
		return err
	}

	return nil
}
