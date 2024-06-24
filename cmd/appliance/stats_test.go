package appliance

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestApplianceStatsCommandJSON(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/stats/appliances",
		httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance.json"),
	)
	defer registry.Teardown()
	registry.Serve()
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	url := fmt.Sprintf("http://localhost:%d", registry.Port)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   url,
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.BaseURL = func() string {
		return url + "/admin"
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
	cmd := NewApplianceCmd(f)
	cmd.SetArgs([]string{"stats", "--json"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}
	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	if !util.IsJSON(string(got)) {
		t.Fatalf("Expected JSON output - got stdout\n%s\n", string(got))
	}
}

func TestApplianceStatsCommandTable(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/stats/appliances",
		httpmock.JSONResponse("../../pkg/appliance/fixtures/stats_appliance.json"),
	)
	defer registry.Teardown()
	registry.Serve()
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	url := fmt.Sprintf("http://localhost:%d", registry.Port)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   url,
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.BaseURL = func() string {
		return url + "/admin"
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
	applianceCMD := NewApplianceCmd(f)
	applianceCMD.SetArgs([]string{"stats", "--include=name=controller&gateway"})
	applianceCMD.SetOut(io.Discard)
	applianceCMD.SetErr(io.Discard)

	_, err := applianceCMD.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	gotStr := string(got)
	want := `Name                                                     Status     Function                 CPU     Memory    Network out/in           Disk    Version    Sessions
----                                                     ------     --------                 ---     ------    --------------           ----    -------    --------
controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1    healthy    LogServer, Controller    0.8%    48.8%     0.26 Kbps / 0.26 Kbps    1.2%    6.2.1      0
gateway-da0375f6-0b28-4248-bd54-a933c4c39008-site1       healthy    Gateway                  0.7%    7.8%      76.8 bps / 96.0 bps      4.9%    6.2.1      5
`
	assert.Equal(t, want, gotStr)
}
