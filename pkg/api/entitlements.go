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


func (s *EntitlementsAPI) NamesMigration(ctx context.Context) (*openapi.EntitlementMigrationInfoList, error) {
	result, response, err := s.API.EntitlementsCloudMigrationsPost(ctx).Execute()
	if err != nil {
		return nil, HTTPErrorResponse(response, err)
	}
	if response.StatusCode >= 400 {
		return nil, HTTPErrorResponse(response, fmt.Errorf("response does not indicate success: %s", response.Status))
	}

	if response.StatusCode == 204 {
		println("Nothing to migrate")
	}
	return result, nil
}
