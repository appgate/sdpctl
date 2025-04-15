package appliance

import (
	"context"
	"errors"
	"fmt"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/prompt"
)

// PromptSelect from online appliances
func PromptSelect(ctx context.Context, a *Appliance, filter map[string]map[string]string, orderBy []string, descending bool) (string, error) {
	appliances, err := a.List(ctx, filter, orderBy, descending)
	if err != nil {
		return "", err
	}
	stats, _, err := a.ApplianceStatus(ctx, nil, orderBy, descending)
	if err != nil {
		return "", err
	}
	appliances, _, err = FilterOnline(appliances, stats.GetData())
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
	selectedIndex, err := prompt.PromptSelectionIndex("select appliance:", names, "")
	if err != nil {
		return "", err
	}

	appliance := appliances[selectedIndex]
	return appliance.GetId(), nil
}

func PromptMultiSelect(ctx context.Context, a *Appliance, filter map[string]map[string]string, orderBy []string, descending bool) ([]string, error) {
	appliances, err := a.List(ctx, filter, orderBy, descending)
	if err != nil {
		return nil, err
	}
	stats, _, err := a.ApplianceStatus(ctx, nil, orderBy, descending)
	if err != nil {
		return nil, err
	}
	appliances, _, err = FilterOnline(appliances, stats.GetData())
	if err != nil {
		return nil, err
	}
	return promptMultiAppliance(appliances)
}

func promptMultiAppliance(appliances []openapi.Appliance) ([]string, error) {
	if len(appliances) == 0 {
		return nil, errors.New("no available options")
	}
	names := []string{}
	for _, a := range appliances {
		names = append(names, fmt.Sprintf("%s - %s - %s", a.GetName(), a.GetSiteName(), a.GetTags()))
	}
	selectedIndices, err := prompt.PromptMultiSelectIndex("select appliance:", names, nil)
	if err != nil {
		return nil, err
	}
	var selectedAppliances []string
	for _, i := range selectedIndices {
		selectedAppliances = append(selectedAppliances, appliances[i].GetId())
	}
	return selectedAppliances, nil
}
