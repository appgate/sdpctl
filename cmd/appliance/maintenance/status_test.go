package maintenance

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/go-cmp/cmp"
)

func TestMaintenanceStatusCommandTable(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/stats/appliances",
		httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
	)
	defer registry.Teardown()
	registry.Serve()
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://127.0.0.1:%d", registry.Port),
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
	cmd := NewStatusCmd(f)

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
	gotStr := string(got)

	want := `Name                                                     Maintenance mode    Details
----                                                     ----------------    -------
controller-da0375f6-0b28-4248-bd54-a933c4c39008-site1    false               Database size is 12 MB
`
	if !cmp.Equal(want, gotStr) {
		t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", gotStr, want)
	}
}
