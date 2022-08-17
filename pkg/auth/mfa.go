package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/api"
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

// ProviderNames HTTP GET /identity-providers/names
func (a *Auth) ProviderNames(ctx context.Context) ([]openapi.IdentityProvidersNamesGet200ResponseDataInner, error) {
	list, response, err := a.APIClient.LoginApi.IdentityProvidersNamesGet(ctx).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	data := list.GetData()
	sort.Slice(data, func(i, j int) bool { return data[i].GetName() < data[j].GetName() })
	return data, nil
}

// Authentication HTTP POST /authentication
func (a *Auth) Authentication(ctx context.Context, opts openapi.LoginRequest) (*openapi.LoginResponse, *MinMax, error) {
	c := a.APIClient
	loginResponse, response, err := c.LoginApi.AuthenticationPost(ctx).LoginRequest(opts).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotAcceptable {
			responseBody, errRead := io.ReadAll(response.Body)
			if errRead != nil {
				return nil, nil, errRead
			}
			errBody := openapi.LoginPost406Response{}
			if err := json.Unmarshal(responseBody, &errBody); err != nil {
				return nil, nil, err
			}
			mm := &MinMax{
				Min: errBody.GetMinSupportedVersion(),
				Max: errBody.GetMaxSupportedVersion(),
			}
			return loginResponse, mm, err
		}
		return nil, nil, api.HTTPErrorResponse(response, err)
	}
	return loginResponse, nil, nil
}

// Authorization HTTP GET /authorization
func (a *Auth) Authorization(ctx context.Context, token string) (*openapi.LoginResponse, error) {
	loginResponse, response, err := a.APIClient.LoginApi.AuthorizationGet(ctx).Authorization(token).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusPreconditionFailed {
			return loginResponse, ErrPreConditionFailed
		}
		return loginResponse, api.HTTPErrorResponse(response, err)
	}
	return loginResponse, nil
}

// InitializeOTP HTTP POST /authentication/otp/initialize
func (a *Auth) InitializeOTP(ctx context.Context, password *string, token string) (*openapi.AuthenticationOtpInitializePost200Response, error) {
	o := openapi.AuthenticationOtpInitializePostRequest{UserPassword: password}
	r, response, err := a.APIClient.LoginApi.AuthenticationOtpInitializePost(ctx).Authorization(token).AuthenticationOtpInitializePostRequest(o).Execute()
	if err != nil {
		return r, api.HTTPErrorResponse(response, err)
	}
	return r, nil
}

var ErrInvalidOneTimePassword = errors.New("Invalid one-time password.")

// PushOTP HTTP POST /authentication/otp
func (a *Auth) PushOTP(ctx context.Context, answer, token string) (*openapi.LoginResponse, error) {
	o := openapi.AuthenticationOtpPostRequest{
		Otp: answer,
	}
	newToken, response, err := a.APIClient.LoginApi.AuthenticationOtpPost(ctx).AuthenticationOtpPostRequest(o).Authorization(token).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusUnauthorized {
			return newToken, ErrInvalidOneTimePassword
		}
		return nil, api.HTTPErrorResponse(response, err)
	}
	return newToken, nil
}
