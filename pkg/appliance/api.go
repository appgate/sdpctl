package appliance

import (
	"context"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

// GetAllAppliances from the appgate sdp collective, without any filter.
func GetAllAppliances(ctx context.Context, client *openapi.APIClient, token string) ([]openapi.Appliance, error) {
	appliances, _, err := client.AppliancesApi.AppliancesGet(ctx).OrderBy("name").Authorization(token).Execute()
	if err != nil {
		return nil, err
	}
	return appliances.GetData(), nil
}

func GetApplianceUpgradeStatus(ctx context.Context, client *openapi.APIClient, token, applianceID string) (openapi.InlineResponse2006, error) {
	status, _, err := client.ApplianceUpgradeApi.AppliancesIdUpgradeGet(ctx, applianceID).Authorization(token).Execute()
	if err != nil {
		return status, err
	}
	return status, nil
}
