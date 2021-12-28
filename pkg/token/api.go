package token

import (
	"context"
	"net/http"

	"github.com/appgate/appgatectl/pkg/api"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

type Token struct {
	APIClient  *openapi.APIClient
	HTTPClient *http.Client
	Token      string
}

func (t *Token) ListDistinguishedNames(ctx context.Context) ([]openapi.DistinguishedName, error) {
	dn, _, err := t.APIClient.ActiveDevicesApi.TokenRecordsDnGet(ctx).Authorization(t.Token).Execute()
	if err != nil {
		return nil, err
	}
	return dn.GetData(), nil
}

func (t *Token) RevokeByDistinguishedName(request openapi.ApiTokenRecordsRevokedByDnDistinguishedNamePutRequest, body openapi.TokenRevocationRequest) (*http.Response, error) {
	_, response, err := request.Authorization(t.Token).TokenRevocationRequest(body).Execute()

	if err != nil {
		httpErr := api.HTTPErrorResponse(response, err)
		if httpErr != nil {
			return nil, httpErr
		}
		return nil, err
	}
	return response, nil
}

func (t *Token) RevokeByTokenType(request openapi.ApiTokenRecordsRevokedByTypeTokenTypePutRequest, body openapi.TokenRevocationRequest) (*http.Response, error) {
	_, response, err := request.Authorization(t.Token).TokenRevocationRequest(body).Execute()

	if err != nil {
		httpErr := api.HTTPErrorResponse(response, err)
		if httpErr != nil {
			return nil, httpErr
		}
		return nil, err
	}
	return response, nil
}

func (t *Token) ReevaluateByDistinguishedName(ctx context.Context, dn string) ([]string, error) {
	reevaluatedDn, _, err := t.APIClient.ActiveDevicesApi.TokenRecordsReevalByDnDistinguishedNamePost(ctx, dn).Authorization(t.Token).Execute()
	if err != nil {
		return nil, err
	}
	return reevaluatedDn.GetReevaluatedDistinguishedNames(), nil
}
