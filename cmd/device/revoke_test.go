package device

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/device"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/stretchr/testify/assert"
)

func setupDeviceRevokeTest(t *testing.T) (*httpmock.Registry, *DeviceOptions, *bytes.Buffer) {
	registry := httpmock.NewRegistry(t)

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
		Debug:  f.Config.Debug,
	}

	return registry, opts, stdout
}

func TestDeviceRevokeByTokenType(t *testing.T) {
	registry, opts, stdout := setupDeviceRevokeTest(t)
	registry.Register("/admin/on-boarded-devices/revoke-tokens", httpmock.JSONResponse("../../pkg/device/fixtures/token_revoke_by_type.json"))
	registry.Serve()
	defer registry.Teardown()

	cmd := NewDeviceRevokeCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--by-token-type", "administration"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	actual, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	expected := `Distinguished Name                                   Device ID                             Username  Provider Name  Device Type  Hostname  Onboarded At                          Last Seen At
------------------                                   ---------                             --------  -------------  -----------  --------  ------------                          ------------
CN=877b2d887c2048e4b2e8daae6bb4c077,CN=bob,OU=local  877b2d88-7c20-48e4-b2e8-daae6bb4c077  bob       local          Client                 2021-12-20 19:29:25.778469 +0000 UTC  2021-12-21 19:29:25.600171 +0000 UTC
CN=37b8c15f300d40b6898c9131019327d4,CN=bob,OU=local  37b8c15f-300d-40b6-898c-9131019327d4  bob       local          Client                 2021-12-20 19:29:22.38033 +0000 UTC   2021-12-21 19:29:22.204692 +0000 UTC
CN=d7d47adbb1bc4a0baaaf9c970d9682c8,CN=bob,OU=local  d7d47adb-b1bc-4a0b-aaaf-9c970d9682c8  bob       local          Client                 2021-12-20 19:29:11.319131 +0000 UTC  2021-12-21 19:29:11.146577 +0000 UTC
CN=9c607025108d4a03b25a22f0d4b2ffba,CN=bob,OU=local  9c607025-108d-4a03-b25a-22f0d4b2ffba  bob       local          Client                 2021-12-20 19:29:07.997339 +0000 UTC  2021-12-21 19:29:07.826791 +0000 UTC
CN=b37de2ed4b4c4d21952f718e2dd6e34b,CN=bob,OU=local  b37de2ed-4b4c-4d21-952f-718e2dd6e34b  bob       local          Client                 2021-12-20 19:29:04.809781 +0000 UTC  2021-12-21 19:29:04.587301 +0000 UTC
CN=f7e1d6fec2344b49b1d65a107025e795,CN=bob,OU=local  f7e1d6fe-c234-4b49-b1d6-5a107025e795  bob       local          Client                 2021-12-20 19:28:56.367895 +0000 UTC  2021-12-21 19:28:55.665852 +0000 UTC
`
	assert.Equal(t, expected, string(actual))
}

func TestDeviceRevokeByTokenTypeJSON(t *testing.T) {
	registry, opts, stdout := setupDeviceRevokeTest(t)
	registry.Register("/admin/on-boarded-devices/revoke-tokens", httpmock.JSONResponse("../../pkg/device/fixtures/token_revoke_by_type.json"))
	registry.Serve()
	defer registry.Teardown()

	opts.useJSON = true
	cmd := NewDeviceRevokeCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--by-token-type", "administration"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	actual, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}

	assert.True(t, util.IsJSON(string(actual)))
}

func TestDeviceRevokeByDistinguishedName(t *testing.T) {
	registry, opts, stdout := setupDeviceRevokeTest(t)
	registry.Register("/admin/on-boarded-devices/revoke-tokens", httpmock.JSONResponse("../../pkg/device/fixtures/token_revoke_by_dn.json"))
	registry.Serve()
	defer registry.Teardown()

	cmd := NewDeviceRevokeCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	actual, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	expected := `Distinguished Name                                     Device ID                             Username  Provider Name  Device Type  Hostname  Onboarded At                          Last Seen At
------------------                                     ---------                             --------  -------------  -----------  --------  ------------                          ------------
CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local  70e07680-1c4b-5bdc-87b4-afc71540e720  admin     local          Client                 2021-12-21 00:27:00.223996 +0000 UTC  2021-12-22 00:27:00.18685 +0000 UTC
CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local  70e07680-1c4b-5bdc-87b4-afc71540e720  admin     local          Client                 2021-12-21 00:27:00.186924 +0000 UTC  2021-12-22 00:27:00.18685 +0000 UTC
CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local  70e07680-1c4b-5bdc-87b4-afc71540e720  admin     local          Client                 2021-12-21 00:24:01.465525 +0000 UTC  2021-12-22 00:24:01.465446 +0000 UTC
`
	assert.Equal(t, expected, string(actual))
}

func TestDeviceRevokeByDistinguishedNameJSON(t *testing.T) {
	registry, opts, stdout := setupDeviceRevokeTest(t)
	registry.Register("/admin/on-boarded-devices/revoke-tokens", httpmock.JSONResponse("../../pkg/device/fixtures/token_revoke_by_dn.json"))
	registry.Serve()
	defer registry.Teardown()

	opts.useJSON = true
	cmd := NewDeviceRevokeCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	actual, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}

	assert.True(t, util.IsJSON(string(actual)))
}

func TestInvalidArgumentsAndOptions(t *testing.T) {
	registry, opts, _ := setupDeviceRevokeTest(t)
	defer registry.Teardown()

	cmd1 := NewDeviceRevokeCmd(opts)
	cmd1.SetOut(io.Discard)
	cmd1.SetErr(io.Discard)
	cmd1.SetArgs([]string{"CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local", "--by-token-type", "administration"})
	_, err1 := cmd1.ExecuteC()
	assert.Equal(t, "Cannot set both <distinguished-name> and --by-token-type", err1.Error())

	cmd2 := NewDeviceRevokeCmd(opts)
	cmd2.SetOut(io.Discard)
	cmd2.SetErr(io.Discard)
	cmd2.SetArgs([]string{})
	_, err2 := cmd2.ExecuteC()
	assert.Equal(t, "Must set either <distinghuished-name> or --by-token-type <type>", err2.Error())

	cmd3 := NewDeviceRevokeCmd(opts)
	cmd3.SetOut(io.Discard)
	cmd3.SetErr(io.Discard)
	cmd3.SetArgs([]string{"--by-token-type", "foo"})
	_, err3 := cmd3.ExecuteC()
	assert.Equal(t, "Unknown token type foo. valid types are { administration, adminclaims, entitlements, claims }", err3.Error())

	cmd4 := NewDeviceRevokeCmd(opts)
	cmd4.SetOut(io.Discard)
	cmd4.SetErr(io.Discard)
	cmd4.SetArgs([]string{"--by-token-type", "foo", "--token-type", "foo"})
	_, err4 := cmd4.ExecuteC()
	assert.Equal(t, "Cannot set --token-type when using --by-token-type <type>", err4.Error())
}
