package device

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

type Device struct {
	APIClient  *openapi.APIClient
	HTTPClient *http.Client
	Token      string
}

func (t *Device) ListDistinguishedNames(ctx context.Context, orderBy []string, descending bool) ([]openapi.OnBoardedDevice, error) {
	dn, response, err := t.APIClient.RegisteredDevicesApi.OnBoardedDevicesGet(ctx).Authorization(t.Token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return orderDeviceList(dn.GetData(), orderBy, descending)
}

func (t *Device) RevokeByDistinguishedName(request openapi.ApiOnBoardedDevicesRevokeTokensPostRequest, body openapi.DeviceRevocationRequest) (*http.Response, error) {
	_, response, err := request.Authorization(t.Token).DeviceRevocationRequest(body).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return response, nil
}

func (t *Device) ReevaluateByDistinguishedName(ctx context.Context, dn string) ([]string, error) {
	reevaluatedDn, response, err := t.APIClient.RegisteredDevicesApi.OnBoardedDevicesReevaluateDistinguishedNamePost(ctx, dn).Authorization(t.Token).Execute()
	if err != nil {
		return nil, api.HTTPErrorResponse(response, err)
	}
	return reevaluatedDn.GetReevaluatedDistinguishedNames(), nil
}

func orderDeviceList(devices []openapi.OnBoardedDevice, orderBy []string, descending bool) ([]openapi.OnBoardedDevice, error) {
	var errs *multierror.Error
	for i := len(orderBy) - 1; i >= 0; i-- {
		switch strings.ToLower(orderBy[i]) {
		case "distinguished-name", "dn":
			sort.SliceStable(devices, func(i, j int) bool { return devices[i].GetDistinguishedName() < devices[j].GetDistinguishedName() })
		case "hostname":
			sort.SliceStable(devices, func(i, j int) bool { return devices[i].GetHostname() < devices[j].GetHostname() })
		case "username", "user":
			sort.SliceStable(devices, func(i, j int) bool { return devices[i].GetUsername() < devices[j].GetUsername() })
		case "provider-name", "provider":
			sort.SliceStable(devices, func(i, j int) bool { return devices[i].GetProviderName() < devices[j].GetProviderName() })
		case "device-id", "device":
			sort.SliceStable(devices, func(i, j int) bool { return devices[i].GetDeviceId() < devices[j].GetDeviceId() })
		case "last-issued":
			sort.SliceStable(devices, func(i, j int) bool { return devices[i].GetLastSeenAt().Before(devices[j].GetLastSeenAt()) })
		default:
			errs = multierror.Append(errs, fmt.Errorf("keyword not sortable: %s", orderBy[i]))
		}
	}

	if descending {
		devices = util.Reverse(devices)
	}

	return devices, errs.ErrorOrNil()
}
