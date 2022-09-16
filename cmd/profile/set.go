package profile

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/spf13/cobra"
)

// NewSetCmd return a new profile set command
func NewSetCmd(opts *commandOpts) *cobra.Command {
	return &cobra.Command{
		Use:     "set",
		Short:   docs.ProfileSetDoc.Short,
		Long:    docs.ProfileSetDoc.Long,
		Example: docs.ProfileSetDoc.ExampleString(),
		RunE: func(c *cobra.Command, args []string) error {
			return setRun(c, args, opts)
		},
	}
}

func setRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !profiles.FileExists() {
		fmt.Fprintln(opts.Out, "no profiles added")
		fmt.Fprintln(opts.Out, "run 'sdpctl profile add' first")
		return nil
	}
	p, err := profiles.Read()
	if err != nil {
		return err
	}
	length := len(p.List)
	list := make([]string, 0, length)
	for _, p := range p.List {
		list = append(list, p.Name)
	}
	index := 0
	if len(args) == 1 {
		found := false
		q := args[0]
		for i, profile := range p.List {
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
			Message:  "select profile:",
			Options:  list,
		}
		if err := prompt.SurveyAskOne(qs, &index, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}
	p.Current = &p.List[index].Directory
	fmt.Fprintf(opts.Out, "%s (%s) is selected as current sdp profile profile\n", p.List[index].Name, p.List[index].Directory)

	if err := profiles.Write(p); err != nil {
		return err
	}
	if !p.CurrentConfigExists() {
		fmt.Fprintf(opts.Out, "%s is not configured yet, run 'sdpctl configure'\n", p.List[index].Name)
	}

	return nil
}
