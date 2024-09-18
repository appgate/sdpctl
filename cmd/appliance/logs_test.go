package appliance

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
)

func TestNewLogsCmd(t *testing.T) {
	dir := t.TempDir()
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/appliances/20e75a08-96c6-4ea3-833e-cdbac346e2ae/logs",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/vnd.appgate.peer-v18+zip")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, string(`binary data`))
			}
		},
	)

	defer registry.Teardown()
	registry.Serve()
	stdout := &bytes.Buffer{}

	stderr := &bytes.Buffer{}
	url := fmt.Sprintf("http://localhost:%d", registry.Port)
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   url,
		},
		IOOutWriter: stdout,
		StdErr:      stderr,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.BaseURL = func() string {
		return url + "/admin"
	}
	f.CustomHTTPClient = func() (*http.Client, error) {
		return &http.Client{}, nil
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
	cmd := NewLogsCmd(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "")
	cmd.PersistentFlags().Bool("descending", false, "")
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.SetArgs([]string{"20e75a08-96c6-4ea3-833e-cdbac346e2ae", "--path", dir})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

}
