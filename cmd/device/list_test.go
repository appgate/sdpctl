package device

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/device"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/stretchr/testify/assert"
)

func setupDeviceListTest(t *testing.T) (*httpmock.Registry, *DeviceOptions, *bytes.Buffer) {
	t.Helper()
	registry := httpmock.NewRegistry(t)
	registry.Register("/admin/on-boarded-devices", httpmock.JSONResponse("../../pkg/device/fixtures/device_list.json"))
	registry.Serve()

	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d/admin", registry.Port),
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
	}

	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.Device = func(c *configuration.Config) (*device.Device, error) {
		api, _ := f.APIClient(c)
		device := &device.Device{
			APIClient:  api,
			HTTPClient: api.GetConfig().HTTPClient,
			Token:      "",
		}
		return device, nil
	}

	opts := &DeviceOptions{
		Config: f.Config,
		Out:    f.IOOutWriter,
		Device: f.Device,
	}

	return registry, opts, stdout
}

func TestDeviceList(t *testing.T) {
	registry, opts, out := setupDeviceListTest(t)
	defer registry.Teardown()

	cmd := NewDeviceListCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}
	actual, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}

	expected := `Distinguished Name                                       Device ID                               Username    Provider Name    Device Type    Hostname           Onboarded At                            Last Seen At
------------------                                       ---------                               --------    -------------    -----------    --------           ------------                            ------------
CN=3332396333654131b235633864363134,CN=admin,OU=local    33323963-3365-4131-b235-633864363134    admin       local            Client         user14.test.com    2021-12-19 19:29:29.107201 +0000 UTC    2021-12-20 18:37:46.634198 +0000 UTC
CN=37b8c15f300d40b6898c9131019327d4,CN=bob,OU=local      37b8c15f-300d-40b6-898c-9131019327d4    bob         local            Client         user4.test.com     2021-12-09 19:29:29.107201 +0000 UTC    2021-12-20 19:29:22.38033 +0000 UTC
CN=43f87ebf811249f8a3a965edc7db0601,CN=bob,OU=local      43f87ebf-8112-49f8-a3a9-65edc7db0601    bob         local            Client         user6.test.com     2021-12-11 19:29:29.107201 +0000 UTC    2021-12-20 19:29:14.519041 +0000 UTC
CN=522db03c06494122a508befb91ba95af,CN=bob,OU=local      522db03c-0649-4122-a508-befb91ba95af    bob         local            Client         user5.test.com     2021-12-10 19:29:29.107201 +0000 UTC    2021-12-20 19:29:17.801181 +0000 UTC
CN=55e3408fe69c49358d6f345e3d2ee4bd,CN=admin,OU=local    55e3408f-e69c-4935-8d6f-345e3d2ee4bd    admin       local            Client         user2.test.com     2021-12-07 19:29:29.107201 +0000 UTC    2021-12-20 19:29:27.451869 +0000 UTC
CN=6633333637304266a631306131646637,CN=admin,OU=local    66333336-3730-4266-a631-306131646637    admin       local            Client         user12.test.com    2021-12-17 19:29:29.107201 +0000 UTC    2021-12-20 19:25:29.414827 +0000 UTC
CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local    70e07680-1c4b-5bdc-87b4-afc71540e720    admin       local            Client         user13.test.com    2021-12-18 19:29:29.107201 +0000 UTC    2021-12-20 19:24:34.652187 +0000 UTC
CN=877b2d887c2048e4b2e8daae6bb4c077,CN=bob,OU=local      877b2d88-7c20-48e4-b2e8-daae6bb4c077    bob         local            Client         user3.test.com     2021-12-08 19:29:29.107201 +0000 UTC    2021-12-20 19:29:25.778469 +0000 UTC
CN=9c607025108d4a03b25a22f0d4b2ffba,CN=bob,OU=local      9c607025-108d-4a03-b25a-22f0d4b2ffba    bob         local            Client         user8.test.com     2021-12-13 19:29:29.107201 +0000 UTC    2021-12-20 19:29:07.997339 +0000 UTC
CN=a3c70825dc0945c48dbc6a6d991d7d0b,CN=bob,OU=local      a3c70825-dc09-45c4-8dbc-6a6d991d7d0b    bob         local            Client         user10.test.com    2021-12-15 19:29:29.107201 +0000 UTC    2021-12-20 19:29:01.415945 +0000 UTC
CN=b37de2ed4b4c4d21952f718e2dd6e34b,CN=bob,OU=local      b37de2ed-4b4c-4d21-952f-718e2dd6e34b    bob         local            Client         user9.test.com     2021-12-14 19:29:29.107201 +0000 UTC    2021-12-20 19:29:04.809781 +0000 UTC
CN=d7d47adbb1bc4a0baaaf9c970d9682c8,CN=bob,OU=local      d7d47adb-b1bc-4a0b-aaaf-9c970d9682c8    bob         local            Client         user7.test.com     2021-12-12 19:29:29.107201 +0000 UTC    2021-12-20 19:29:11.319131 +0000 UTC
CN=f0f4305444d24991b070160de7b69fe9,CN=bob,OU=local      f0f43054-44d2-4991-b070-160de7b69fe9    bob         local            Client         user1.test.com     2021-12-06 19:29:29.107201 +0000 UTC    2021-12-20 19:29:29.107201 +0000 UTC
CN=f7e1d6fec2344b49b1d65a107025e795,CN=bob,OU=local      f7e1d6fe-c234-4b49-b1d6-5a107025e795    bob         local            Client         user11.test.com    2021-12-16 19:29:29.107201 +0000 UTC    2021-12-20 19:28:56.367895 +0000 UTC
`
	assert.Equal(t, expected, string(actual))
}

func TestDeviceListJSON(t *testing.T) {
	registry, opts, out := setupDeviceListTest(t)
	defer registry.Teardown()

	opts.useJSON = true

	cmd := NewDeviceListCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}
	actual, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}

	assert.True(t, util.IsJSON(string(actual)))
}
