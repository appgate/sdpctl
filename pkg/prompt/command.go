package prompt

import (
	"context"

	"github.com/AlecAivazis/survey/v2"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
)

type AppliancePrompt interface {
	ResolveAppliance(c *configuration.Config) (*appliancepkg.Appliance, error)
}

func SelectAppliance(ctx context.Context, opts AppliancePrompt, config *configuration.Config, filter map[string]map[string]string) (string, error) {
	// Command accepts only one argument
	a, err := opts.ResolveAppliance(config)
	if err != nil {
		return "", err
	}

	appliances, err := a.List(ctx, filter)
	if err != nil {
		return "", err
	}
	stats, _, err := a.Stats(ctx)
	if err != nil {
		return "", err
	}
	appliances, _, err = appliancepkg.FilterAvailable(appliances, stats.GetData())
	if err != nil {
		return "", err
	}

	names := []string{}
	for _, a := range appliances {
		names = append(names, a.GetName())
	}
	qs := &survey.Select{
		PageSize: len(appliances),
		Message:  "select appliance:",
		Options:  names,
	}
	selectedIndex := 0
	survey.AskOne(qs, &selectedIndex, survey.WithValidator(survey.Required))
	appliance := appliances[selectedIndex]
	return appliance.GetId(), nil
}
