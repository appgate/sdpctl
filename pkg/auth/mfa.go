package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/appgate/appgatectl/pkg/api"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

type Auth struct {
	APIClient *openapi.APIClient
}

type MinMax struct {
	Min, Max int32
}

func NewAuth(APIClient *openapi.APIClient) *Auth {
	return &Auth{APIClient: APIClient}
}

var ErrPreConditionFailed = errors.New("OTP required")

func (a *Auth) ProviderNames(ctx context.Context) ([]string, error) {
	result := make([]string, 0)
	list, response, err := a.APIClient.LoginApi.IdentityProvidersNamesGet(ctx).Execute()
	if err != nil {
		httpErr := api.HTTPErrorResponse(response, err)
		if httpErr != nil {
			return nil, httpErr
		}
		return nil, err
	}
	for _, i := range list.GetData() {
		result = append(result, i.GetName())
	}
	return result, nil
}

func (a *Auth) Authentication(ctx context.Context, opts openapi.LoginRequest) (*openapi.LoginResponse, *MinMax, error) {
	c := a.APIClient
	loginResponse, response, err := c.LoginApi.AuthenticationPost(ctx).LoginRequest(opts).Execute()
	if err != nil {
		// TODO use this once we have updated the SDK
		// if err, ok := err.(openapi.GenericOpenAPIError); ok {
		// 	if model, ok := err.Model().(openapi.InlineResponse406); ok {
		// 		fmt.Printf("\n\n err AAA22222 %v\n", err)
		// 		mm := &MinMax{
		// 			Min: model.GetMinSupportedVersion(),
		// 			Max: model.GetMaxSupportedVersion(),
		// 		}
		// 		return &loginResponse, mm, err
		// 	}
		// }
		if response != nil && response.StatusCode == http.StatusNotAcceptable {
			responseBody, errRead := io.ReadAll(response.Body)
			if errRead != nil {
				return nil, nil, errRead
			}
			errBody := openapi.InlineResponse406{}
			if err := json.Unmarshal(responseBody, &errBody); err != nil {
				return nil, nil, err
			}
			mm := &MinMax{
				Min: errBody.GetMinSupportedVersion(),
				Max: errBody.GetMaxSupportedVersion(),
			}
			return &loginResponse, mm, err
		}
		if response != nil && response.StatusCode == http.StatusPreconditionFailed {
			return &loginResponse, nil, ErrPreConditionFailed
		}
		httpErr := api.HTTPErrorResponse(response, err)
		if httpErr != nil {
			return nil, nil, httpErr
		}
		return nil, nil, err
	}
	return &loginResponse, nil, nil
}

func (a *Auth) Authorization(ctx context.Context, password, token string) (*openapi.LoginResponse, error) {
	loginResponse, response, err := a.APIClient.LoginApi.AuthorizationGet(ctx).Authorization(token).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusPreconditionFailed {
			return &loginResponse, ErrPreConditionFailed
		}
		httpErr := api.HTTPErrorResponse(response, err)
		if httpErr != nil {
			return nil, httpErr
		}
		return &loginResponse, err
	}
	return &loginResponse, nil
}

func (a *Auth) InitializeOTP(ctx context.Context, password, token string) (openapi.InlineResponse2007, error) {
	o := openapi.InlineObject7{UserPassword: openapi.PtrString(password)}
	r, response, err := a.APIClient.LoginApi.AuthenticationOtpInitializePost(ctx).Authorization(token).InlineObject7(o).Execute()
	if err != nil {
		httpErr := api.HTTPErrorResponse(response, err)
		if httpErr != nil {
			return r, httpErr
		}
		return r, err
	}
	return r, nil
}

func (a *Auth) PushOTP(ctx context.Context, answer, token string) (*openapi.LoginResponse, error) {
	o := openapi.InlineObject6{
		Otp: answer,
	}
	newToken, response, err := a.APIClient.LoginApi.AuthenticationOtpPost(ctx).InlineObject6(o).Authorization(token).Execute()
	if err != nil {
		httpErr := api.HTTPErrorResponse(response, err)
		if httpErr != nil {
			return nil, httpErr
		}
		return nil, fmt.Errorf("invalid OTP %w", err)
	}
	return &newToken, nil
}
