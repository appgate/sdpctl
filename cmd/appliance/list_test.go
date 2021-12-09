package appliance

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/go-cmp/cmp"
)

func TestApplianceListCommandJSON(t *testing.T) {
	registery := httpmock.NewRegistry()
	registery.Register(
		"/appliances",
		httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
	)
	defer registery.Teardown()
	registery.Serve()
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registery.Port),
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registery.Client, nil
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
	cmd := NewListCmd(f)
	cmd.SetArgs([]string{"--json"})
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
func TestApplianceListCommandTable(t *testing.T) {
	registery := httpmock.NewRegistry()
	registery.Register(
		"/appliances",
		httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
	)
	defer registery.Teardown()
	registery.Serve()
	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registery.Port),
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registery.Client, nil
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
	cmd := NewListCmd(f)

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
	want := `Name                                                         Hostname                 Site                Activated
controller-da0375f6-0b28-4248-bd54-a933c4c39008-site1        controller.devops        Default Site        true
gateway-da0375f6-0b28-4248-bd54-a933c4c39008-site1           gateway.devops           Default Site        true
`
	if !cmp.Equal(want, gotStr) {
		t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", gotStr, want)
	}
}
