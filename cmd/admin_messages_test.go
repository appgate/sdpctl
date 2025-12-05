package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/go-cmp/cmp"
)

func TestAdminMessageCommandOutput(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/admin-messages/summarize",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/vnd.appgate.peer-v19+json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, `{
					"data": [
						{
							"category": "Configuration",
							"count": 2,
							"created": "2023-07-18T08:48:01.354845Z",
							"id": "e2689a4c-88c5-49ce-b257-436027459266",
							"level": "Information",
							"message": "Appliance 'second-controller' has joined the collective. This Appliance operates as a spare.",
							"source": "primary-controller",
							"sources": [
								"primary-controller"
							]
						}
					]
				}`)
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
	cmd := NewAdminMessageCmd(f)

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
	want := `Messages:
Information from primary-controller
2023-07-18 08:48:01.354845 +0000 UTC - 2 occurrences
Appliance 'second-controller' has joined the collective. This Appliance operates as a spare.

`
	if !cmp.Equal(want, gotStr) {
		t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", gotStr, want)
	}
}
