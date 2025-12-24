package api

import (
	"context"
	"fmt"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
)

type EntitlementsAPI struct {
	API   *openapi.EntitlementsApiService
	Token string
}

func (s *EntitlementsAPI) NamesMigration(ctx context.Context, dryRun bool) (*openapi.EntitlementMigrationInfoList, error) {
	result, response, err := s.API.EntitlementsCloudMigrationsPost(ctx).DryRun(dryRun).Execute()
	if err != nil {
		return nil, HTTPErrorResponse(response, err)
	}
	if response.StatusCode >= 400 {
		return nil, HTTPErrorResponse(response, fmt.Errorf("response does not indicate success: %s", response.Status))
	}

	return result, nil
}
