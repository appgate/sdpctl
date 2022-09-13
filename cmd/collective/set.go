package collective

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/spf13/cobra"
)

// NewSetCmd return a new collective set command
func NewSetCmd(opts *commandOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "",
		Long:  "",
		RunE: func(c *cobra.Command, args []string) error {
			return setRun(c, args, opts)
		},
	}
}

func setRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !configuration.ProfileFileExists() {
		fmt.Fprintln(opts.Out, "no profiles added")
		fmt.Fprintln(opts.Out, "run 'sdpctl collective add' first")
		return nil
	}
	profiles, err := configuration.ReadProfiles()
	if err != nil {
		return err
	}
	length := len(profiles.List)
	list := make([]string, 0, length)
	for _, p := range profiles.List {
		list = append(list, p.Name)
	}
	index := 0
	if len(args) == 1 {
		found := false
		q := args[0]
		for i, profile := range profiles.List {
			if q == profile.Name {
				index = i
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Profile %s not found in %v", q, list)
		}
	} else {
		qs := &survey.Select{
			PageSize: length,
			Message:  "select collective:",
			Options:  list,
		}
		if err := prompt.SurveyAskOne(qs, &index, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}
	profiles.Current = &profiles.List[index].Directory
	fmt.Fprintf(opts.Out, "%s (%s) is selected as current sdp collective profile\n", profiles.List[index].Name, profiles.List[index].Directory)

	if err := configuration.WriteProfiles(profiles); err != nil {
		return err
	}
	if !profiles.CurrentConfigExists() {
		fmt.Fprintf(opts.Out, "%s is not configured yet, run 'sdpctl configure'\n", profiles.List[index].Name)
	}

	return nil
}
