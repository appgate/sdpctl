package serviceusers

import (
	"context"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/api"
)

type ServiceUsersAPI struct {
	client *openapi.ServiceUsersApiService
	token  string
}

func NewServiceUsersAPI(client *openapi.ServiceUsersApiService, token string) *ServiceUsersAPI {
	return &ServiceUsersAPI{
		client: client,
		token:  token,
	}
}

func (su *ServiceUsersAPI) List(ctx context.Context) ([]openapi.ServiceUser, error) {
	list, response, err := su.client.ServiceUsersGet(ctx).Authorization(su.token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return list.GetData(), nil
}

func (su *ServiceUsersAPI) Create(ctx context.Context, userData openapi.ServiceUser) (*openapi.ServiceUser, error) {
	result, response, err := su.client.ServiceUsersPost(ctx).ServiceUser(userData).Authorization(su.token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return result, nil
}

func (su *ServiceUsersAPI) Read(ctx context.Context, id string) (*openapi.ServiceUser, error) {
	user, response, err := su.client.ServiceUsersIdGet(ctx, id).Authorization(su.token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return user, nil
}

func (su *ServiceUsersAPI) Update(ctx context.Context, userData openapi.ServiceUser) (*openapi.ServiceUser, error) {
	user, response, err := su.client.ServiceUsersIdPut(ctx, userData.GetId()).ServiceUser(userData).Authorization(su.token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return user, nil
}

func (su *ServiceUsersAPI) Delete(ctx context.Context, id string) error {
	response, err := su.client.ServiceUsersIdDelete(ctx, id).Authorization(su.token).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	return nil
}
