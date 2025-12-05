package appliance

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/google/go-cmp/cmp"
)

func TestApplianceListCommandJSON(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/appliances",
		httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
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
	cmd := NewListCmd(f)
	cmd.PersistentFlags().StringToStringP("include", "i", map[string]string{}, "")
	cmd.PersistentFlags().StringToStringP("exclude", "e", map[string]string{}, "")
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.PersistentFlags().Bool("descending", false, "")
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
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/appliances",
		httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
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
			URL:   fmt.Sprintf("http://appgate.test:%d", registry.Port),
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
	cmd := NewListCmd(f)
	cmd.PersistentFlags().StringToStringP("include", "i", map[string]string{}, "")
	cmd.PersistentFlags().StringToStringP("exclude", "e", map[string]string{}, "")
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.PersistentFlags().Bool("descending", false, "")
	cmd.SetArgs([]string{})
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
	want := `Name                                                     ID                                      Hostname          Site            Activated
----                                                     --                                      --------          ----            ---------
controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1    4c07bc67-57ea-42dd-b702-c2d6c45419fc    appgate.test      Default Site    true
gateway-da0375f6-0b28-4248-bd54-a933c4c39008-site1       ee639d70-e075-4f01-596b-930d5f24f569    gateway.devops    Default Site    true
`
	if !cmp.Equal(want, gotStr) {
		t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", gotStr, want)
	}
}

func TestApplianceFiltering(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/appliances",
		httpmock.JSONResponse("../../pkg/appliance/fixtures/appliance_list.json"),
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

	// Need to call parent command since list command inherits the filter flag from it
	cmd := NewApplianceCmd(f)
	cmd.SetArgs([]string{"list", "--include=name=controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1"})
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
	want := `Name                                                     ID                                      Hostname        Site            Activated
----                                                     --                                      --------        ----            ---------
controller-4c07bc67-57ea-42dd-b702-c2d6c45419fc-site1    4c07bc67-57ea-42dd-b702-c2d6c45419fc    appgate.test    Default Site    true
`
	if !cmp.Equal(want, gotStr) {
		t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", gotStr, want)
	}
}
