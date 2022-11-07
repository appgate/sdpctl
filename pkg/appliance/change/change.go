package change

import (
	"context"
	"errors"
	"fmt"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/cenkalti/backoff/v4"
)

type ApplianceChange struct {
	APIClient *openapi.APIClient
	Token     string
}

// Get HTTP GET /appliances/{id}/change/{changeId}
func (ac *ApplianceChange) Get(ctx context.Context, changeID, applianceID string) (*openapi.AppliancesIdChangeChangeIdGet200Response, error) {
	result, response, err := ac.APIClient.ApplianceChangeApi.AppliancesIdChangeChangeIdGet(ctx, changeID, applianceID).Authorization(ac.Token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return result, nil
}

// RetryUntilCompleted is a blocking function that retries appliance change until it has reached desired result success
func (ac *ApplianceChange) RetryUntilCompleted(ctx context.Context, changeID, applianceID string) (*openapi.AppliancesIdChangeChangeIdGet200Response, error) {
	var changeResult *openapi.AppliancesIdChangeChangeIdGet200Response
	err := backoff.Retry(func() error {
		change, err := ac.Get(ctx, changeID, applianceID)
		if err != nil {
			return err
		}
		if change.GetStatus() == "running" {
			return errors.New("Change is still running, retry")
		}
		if change.GetResult() == "failure" {
			if v, ok := change.GetDetailsOk(); ok && len(*v) > 0 {
				return backoff.Permanent(fmt.Errorf("unable to apply on appliance id %s change %s", applianceID, *v))
			}
			return backoff.Permanent(fmt.Errorf("appliance change failed on appliance id %s %s %s", applianceID, change.GetResult(), change.GetDetails()))
		}
		if change.GetResult() == "failure" {
			if v, ok := change.GetDetailsOk(); ok && len(*v) > 0 {
				return backoff.Permanent(fmt.Errorf("unable to apply change %s", *v))
			}
			return backoff.Permanent(fmt.Errorf("appliance change failed %s %s", change.GetResult(), change.GetDetails()))
		}
		if change.GetResult() != "success" && change.GetStatus() == "completed" {
			return fmt.Errorf("Got result %s and status %s", change.GetResult(), change.GetStatus())
		}
		changeResult = change
		return nil
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))

	return changeResult, err
}
