package token

import (
	"bytes"
	"fmt"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/appgatectl/pkg/token"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func setupTokenRevokeTest() (*httpmock.Registry, *RevokeOptions, *bytes.Buffer) {
	registry := httpmock.NewRegistry()
	registry.Register("/token-records/revoked/by-dn/CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local", httpmock.JSONResponse("../../pkg/token/fixtures/token_revoke_by_dn.json"))
	registry.Register("/token-records/revoked/by-type/administration", httpmock.JSONResponse("../../pkg/token/fixtures/token_revoke_by_type.json"))
	registry.Serve()

	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
	}

	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.Token = func(c *configuration.Config) (*token.Token, error) {
		api, _ := f.APIClient(c)
		token := &token.Token{
			APIClient:  api,
			HTTPClient: api.GetConfig().HTTPClient,
			Token:      "",
		}
		return token, nil
	}

	opts := &RevokeOptions{
        TokenOptions: &TokenOptions{
            Config: f.Config,
            Out:    f.IOOutWriter,
            Token:  f.Token,
            Debug:  f.Config.Debug,
        },
    }

	return registry, opts, stdout
}

func TestTokenRevokeByTokenType(t *testing.T) {
	registry, opts, stdout := setupTokenRevokeTest()
	defer registry.Teardown()

	cmd := NewTokenRevokeByTokenTypeCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"administration"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	actual, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	expected := `ID                                    Type         Distinguished Name                                   Issued                                Expires                               Revoked  Site ID                               Site Name          Revocation Time                       Device ID                             Username  Provider Name  Controller Hostname
--                                    ----         ------------------                                   ------                                -------                               -------  -------                               ---------          ---------------                       ---------                             --------  -------------  -------------------
b7dfa30a-885c-4ac6-ba4e-e9cca36709a9  Entitlement  CN=877b2d887c2048e4b2e8daae6bb4c077,CN=bob,OU=local  2021-12-20 19:29:25.778469 +0000 UTC  2021-12-21 19:29:25.600171 +0000 UTC  true     46507910-c5a2-4d83-b405-56e25d323770  simple_setup Site  2021-12-21 00:29:29.824619 +0000 UTC  877b2d88-7c20-48e4-b2e8-daae6bb4c077  bob       local          envy-10-97-146-2.devops
3563737d-af6d-48d1-aaac-3531fb250de1  Entitlement  CN=37b8c15f300d40b6898c9131019327d4,CN=bob,OU=local  2021-12-20 19:29:22.38033 +0000 UTC   2021-12-21 19:29:22.204692 +0000 UTC  true     46507910-c5a2-4d83-b405-56e25d323770  simple_setup Site  2021-12-21 00:29:29.824619 +0000 UTC  37b8c15f-300d-40b6-898c-9131019327d4  bob       local          envy-10-97-146-2.devops
7ba87ab9-b06f-4256-af3d-d7ab7ede9783  Entitlement  CN=d7d47adbb1bc4a0baaaf9c970d9682c8,CN=bob,OU=local  2021-12-20 19:29:11.319131 +0000 UTC  2021-12-21 19:29:11.146577 +0000 UTC  true     46507910-c5a2-4d83-b405-56e25d323770  simple_setup Site  2021-12-21 00:29:29.824619 +0000 UTC  d7d47adb-b1bc-4a0b-aaaf-9c970d9682c8  bob       local          envy-10-97-146-2.devops
8cf66b11-8f19-4ac5-ae25-4dae82a85a5b  Entitlement  CN=9c607025108d4a03b25a22f0d4b2ffba,CN=bob,OU=local  2021-12-20 19:29:07.997339 +0000 UTC  2021-12-21 19:29:07.826791 +0000 UTC  true     46507910-c5a2-4d83-b405-56e25d323770  simple_setup Site  2021-12-21 00:29:29.824619 +0000 UTC  9c607025-108d-4a03-b25a-22f0d4b2ffba  bob       local          envy-10-97-146-2.devops
e433ec11-cb89-4a59-a0e3-f89848783b04  Entitlement  CN=b37de2ed4b4c4d21952f718e2dd6e34b,CN=bob,OU=local  2021-12-20 19:29:04.809781 +0000 UTC  2021-12-21 19:29:04.587301 +0000 UTC  true     46507910-c5a2-4d83-b405-56e25d323770  simple_setup Site  2021-12-21 00:29:29.824619 +0000 UTC  b37de2ed-4b4c-4d21-952f-718e2dd6e34b  bob       local          envy-10-97-146-2.devops
381d72ad-3910-48d8-9fef-84c7b94d1982  Entitlement  CN=f7e1d6fec2344b49b1d65a107025e795,CN=bob,OU=local  2021-12-20 19:28:56.367895 +0000 UTC  2021-12-21 19:28:55.665852 +0000 UTC  true     46507910-c5a2-4d83-b405-56e25d323770  simple_setup Site  2021-12-21 00:29:29.824619 +0000 UTC  f7e1d6fe-c234-4b49-b1d6-5a107025e795  bob       local          envy-10-97-146-2.devops
`
	assert.Equal(t, string(actual), expected)
}

func TestTokenRevokeByTokenTypeJSON(t *testing.T) {
	registry, opts, stdout := setupTokenRevokeTest()
	defer registry.Teardown()

	opts.TokenOptions.useJSON = true
	cmd := NewTokenRevokeByTokenTypeCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"administration"})

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

func TestTokenRevokeByDistinguishedName(t *testing.T) {
	registry, opts, stdout := setupTokenRevokeTest()
	defer registry.Teardown()

	cmd := NewTokenRevokeByDistinguishedNameCmd(opts)
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
	expected := `ID                                    Type            Distinguished Name                                     Issued                                Expires                               Revoked  Site ID  Site Name  Revocation Time                       Device ID                             Username  Provider Name  Controller Hostname
--                                    ----            ------------------                                     ------                                -------                               -------  -------  ---------  ---------------                       ---------                             --------  -------------  -------------------
d7559914-a77e-411e-af71-60498d5ccddd  Administration  CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local  2021-12-21 00:27:00.223996 +0000 UTC  2021-12-22 00:27:00.18685 +0000 UTC   true                         2021-12-21 00:32:20.933517 +0000 UTC  70e07680-1c4b-5bdc-87b4-afc71540e720  admin     local          envy-10-97-146-2.devops
c3eeb87b-e406-42ee-a1a9-0dadb4752023  AdminClaims     CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local  2021-12-21 00:27:00.186924 +0000 UTC  2021-12-22 00:27:00.18685 +0000 UTC   true                         2021-12-21 00:32:20.933517 +0000 UTC  70e07680-1c4b-5bdc-87b4-afc71540e720  admin     local          envy-10-97-146-2.devops
43c9c2bb-3e31-4ce3-a779-8d33427f91ae  AdminClaims     CN=70e076801c4b5bdc87b4afc71540e720,CN=admin,OU=local  2021-12-21 00:24:01.465525 +0000 UTC  2021-12-22 00:24:01.465446 +0000 UTC  true                         2021-12-21 00:32:20.933517 +0000 UTC  70e07680-1c4b-5bdc-87b4-afc71540e720  admin     local          envy-10-97-146-2.devops
`
	assert.Equal(t, string(actual), expected)
}

func TestTokenRevokeByDistinguishedNameJOSN(t *testing.T) {
	registry, opts, stdout := setupTokenRevokeTest()
	defer registry.Teardown()

	opts.TokenOptions.useJSON = true
	cmd := NewTokenRevokeByDistinguishedNameCmd(opts)
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
