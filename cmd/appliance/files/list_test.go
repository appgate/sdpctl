package files

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/stretchr/testify/assert"
)

func setupListTest(t *testing.T) (*httpmock.Registry, *factory.Factory, *bytes.Buffer) {
	t.Helper()
	registry := httpmock.NewRegistry(t)
	registry.Register("/files", httpmock.JSONResponse("../../../pkg/appliance/fixtures/file_list.json"))
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
	f.Appliance = func(c *configuration.Config) (*appliance.Appliance, error) {
		api, _ := f.APIClient(c)

		a := &appliance.Appliance{
			APIClient:  api,
			HTTPClient: api.GetConfig().HTTPClient,
			Token:      "",
		}
		return a, nil
	}

	return registry, f, stdout
}

func TestFilesList(t *testing.T) {
	registry, f, out := setupListTest(t)
	defer registry.Teardown()

	cmd := NewFilesCmd(f)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"list"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}
	actual, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}

	expect := `Name                                Status    Created                                 Modified                                Failure Reason
----                                ------    -------                                 --------                                --------------
appgate-6.0.1-29983-beta.img.zip    Failed    2022-08-18 11:25:52.494572 +0000 UTC    2022-08-18 11:25:52.494572 +0000 UTC    401 Unauthorized
appgate-5.5.1-29983.img.zip         Ready     2022-08-18 11:26:52.494572 +0000 UTC    2022-08-18 12:25:52.494572 +0000 UTC    
`

	assert.Equal(t, string(actual), expect)
}

func TestFilesListJSON(t *testing.T) {
	registry, f, out := setupListTest(t)
	defer registry.Teardown()

	cmd := NewFilesCmd(f)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"list", "--json"})

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
