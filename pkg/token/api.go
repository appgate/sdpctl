package token

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
)

type Token struct {
	APIClient  *openapi.APIClient
	HTTPClient *http.Client
	Token      string
}

func (t *Token) ListDistinguishedNames(ctx context.Context, orderBy []string, descending bool) ([]openapi.OnBoardedDevice, error) {
	dn, response, err := t.APIClient.RegisteredDevicesApi.OnBoardedDevicesGet(ctx).Authorization(t.Token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return orderTokenList(dn.GetData(), orderBy, descending)
}

func (t *Token) RevokeByDistinguishedName(request openapi.ApiOnBoardedDevicesRevokeTokensPostRequest, body openapi.DeviceRevocationRequest) (*http.Response, error) {
	_, response, err := request.Authorization(t.Token).DeviceRevocationRequest(body).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return response, nil
}

func (t *Token) ReevaluateByDistinguishedName(ctx context.Context, dn string) ([]string, error) {
	reevaluatedDn, response, err := t.APIClient.ActiveDevicesApi.OnBoardedDevicesReevaluateDistinguishedNamePost(ctx, dn).Authorization(t.Token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return reevaluatedDn.GetReevaluatedDistinguishedNames(), nil
}

func orderTokenList(tokens []openapi.OnBoardedDevice, orderBy []string, descending bool) ([]openapi.OnBoardedDevice, error) {
	var errs *multierror.Error
	for i := len(orderBy) - 1; i >= 0; i-- {
		switch strings.ToLower(orderBy[i]) {
		case "distinguished-name", "dn":
			sort.SliceStable(tokens, func(i, j int) bool { return tokens[i].GetDistinguishedName() < tokens[j].GetDistinguishedName() })
		case "hostname":
			sort.SliceStable(tokens, func(i, j int) bool { return tokens[i].GetHostname() < tokens[j].GetHostname() })
		case "username", "user":
			sort.SliceStable(tokens, func(i, j int) bool { return tokens[i].GetUsername() < tokens[j].GetUsername() })
		case "provider-name", "provider":
			sort.SliceStable(tokens, func(i, j int) bool { return tokens[i].GetProviderName() < tokens[j].GetProviderName() })
		case "device-id", "device":
			sort.SliceStable(tokens, func(i, j int) bool { return tokens[i].GetDeviceId() < tokens[j].GetDeviceId() })
		case "last-issued":
			sort.SliceStable(tokens, func(i, j int) bool { return tokens[i].GetLastSeenAt().Before(tokens[j].GetLastSeenAt()) })
		default:
			errs = multierror.Append(errs, fmt.Errorf("keyword not sortable: %s", orderBy[i]))
		}
	}

	if descending {
		tokens = util.Reverse(tokens)
	}

	return tokens, errs.ErrorOrNil()
}
