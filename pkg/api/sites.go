package api

import (
	"context"
	"fmt"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
)

type SitesAPI struct {
	API   *openapi.SitesApiService
	Token string
}

func (s *SitesAPI) ListSites(ctx context.Context) ([]openapi.Site, error) {
	result, response, err := s.API.SitesGet(ctx).Authorization(s.Token).Execute()
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("response does not indicate success: %s", response.Status)
	}
	return result.GetData(), nil
}

func (s *SitesAPI) ListResources(ctx context.Context, id string, resolver *openapi.ResolverType, type_ *openapi.ResourceType) (*openapi.ResolverResources, error) {
	result, response, err := s.API.SitesIdResourcesGet(ctx, id).Authorization(s.Token).Resolver(*resolver).Type_(*type_).Execute()
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("response does not indicate success: %s", response.Status)
	}
	return result, nil
}

func (s *SitesAPI) SiteStatus(ctx context.Context) ([]openapi.SiteWithStatus, error) {
	result, response, err := s.API.SitesStatusGet(ctx).Authorization(s.Token).Execute()
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("response does not indicate success: %s", response.Status)
	}
	return result.GetData(), nil
}
