package appliance

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
)

func TestMetricCommand(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/appliances/0a11e7ba-4d18-4be1-bdc1-083be1411d7e/metrics",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/plain")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`
# HELP audit_event Total audit event count
# TYPE audit_event counter
audit_event{collective_id="8dda5969-e9de-4d0f-b4e8-38954c7c0507", appliance_id="ecb7d7ed-ec6a-4d39-4271-6b8b785520d3", type="appliance_status_changed"} 3.0
`))
			}
		},
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
	cmd := NewMetricCmd(f)
	cmd.PersistentFlags().Bool("descending", false, "")
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.SetArgs([]string{"0a11e7ba-4d18-4be1-bdc1-083be1411d7e"})
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
	want := `
# HELP audit_event Total audit event count
# TYPE audit_event counter
audit_event{collective_id="8dda5969-e9de-4d0f-b4e8-38954c7c0507", appliance_id="ecb7d7ed-ec6a-4d39-4271-6b8b785520d3", type="appliance_status_changed"} 3.0

`
	if string(got) != want {
		t.Fatalf("want:\n%q\n\nGot:%q\n", want, string(got))
	}
}

func TestMetricCommandSpecificMetric(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/appliances/0a11e7ba-4d18-4be1-bdc1-083be1411d7e/metrics/vpn_total_sessions",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/plain")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(``))
			}
		},
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
	cmd := NewMetricCmd(f)
	cmd.PersistentFlags().Bool("descending", false, "")
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.SetArgs([]string{"0a11e7ba-4d18-4be1-bdc1-083be1411d7e", "vpn_total_sessions"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}
}
