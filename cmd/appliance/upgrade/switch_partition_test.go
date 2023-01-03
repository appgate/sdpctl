package upgrade

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
)

func TestNewSwitchPartitionCmd(t *testing.T) {
	registry := httpmock.NewRegistry(t)

	registry.Register(
		"/appliances/20e75a08-96c6-4ea3-833e-cdbac346e2ae",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "name": "a gateway"
                }`))
			}
		},
	)
	registry.Register(
		"/appliances/20e75a08-96c6-4ea3-833e-cdbac346e2ae/upgrade/switch-partition",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "id": "aef6c5aa-a7fc-43c2-bda8-d3e6b4523aeb"
                }`))
			}
		},
	)
	registry.Register(
		"/appliances/20e75a08-96c6-4ea3-833e-cdbac346e2ae/change/aef6c5aa-a7fc-43c2-bda8-d3e6b4523aeb",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`{
                    "details": "Invalid event in state appliance_ready",
                    "id": "aef6c5aa-a7fc-43c2-bda8-d3e6b4523aeb",
                    "result": "failure",
                    "status": "completed"
                }`))
			}
		},
	)

	defer registry.Teardown()
	registry.Serve()
	stdout := &bytes.Buffer{}

	stderr := &bytes.Buffer{}
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
		},
		IOOutWriter: stdout,
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
	cmd := NewSwitchPartitionCmd(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "")
	cmd.SetArgs([]string{"20e75a08-96c6-4ea3-833e-cdbac346e2ae"})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

}
