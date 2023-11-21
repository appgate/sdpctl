package appliance

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	"github.com/appgate/sdpctl/pkg/prompt"
)

// PromptSelect from online appliances
func PromptSelect(ctx context.Context, a *Appliance, filter map[string]map[string]string, orderBy []string, descending bool) (string, error) {
	appliances, err := a.List(ctx, filter, orderBy, descending)
	if err != nil {
		return "", err
	}
	stats, _, err := a.Stats(ctx, nil, orderBy, descending)
	if err != nil {
		return "", err
	}
	appliances, _, err = FilterAvailable(appliances, stats.GetData())
	if err != nil {
		return "", err
	}
	return promptAppliance(appliances)
}

// PromptSelectAll from all appliances, offline and online
func PromptSelectAll(ctx context.Context, a *Appliance, filter map[string]map[string]string, orderBy []string, descending bool) (string, error) {
	appliances, err := a.List(ctx, filter, orderBy, descending)
	if err != nil {
		return "", err
	}
	return promptAppliance(appliances)
}

func promptAppliance(appliances []openapi.Appliance) (string, error) {
	if len(appliances) == 0 {
		return "", errors.New("no available options")
	}
	names := []string{}
	for _, a := range appliances {
		names = append(names, fmt.Sprintf("%s - %s - %s", a.GetName(), a.GetSiteName(), a.GetTags()))
	}
	qs := &survey.Select{
		PageSize: len(appliances),
		Message:  "select appliance:",
		Options:  names,
	}
	selectedIndex := 0
	if err := prompt.SurveyAskOne(qs, &selectedIndex, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	appliance := appliances[selectedIndex]
	return appliance.GetId(), nil
}
