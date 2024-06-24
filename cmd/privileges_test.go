package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/go-cmp/cmp"
)

var adminResponse = `
{
    "expires": "2023-06-14T14:48:35.333918184Z",
    "token": "myToken",
    "user": {
        "canAccessAuditLogs": false,
        "name": "admin",
        "needTwoFactorAuth": false,
        "privileges": [
            {
                "scope": {
                    "all": true,
                    "ids": [],
                    "tags": []
                },
                "target": "All",
                "type": "All"
            }
        ]
    }
}`

func TestNewPrivilegesCmd(t *testing.T) {
	registry := httpmock.NewRegistry(t)
	registry.Register(
		"/admin/authorization",
		func(rw http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				fmt.Fprint(rw, adminResponse)
			}
		},
	)
	defer registry.Teardown()
	registry.Serve()

	stdout := &bytes.Buffer{}
	f := &factory.Factory{
		Config: &configuration.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registry.Port),
		},
		IOOutWriter: stdout,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	cmd := NewPrivilegesCmd(f)

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
	want := `
admin have the following privileges

target    type    scope
------    ----    -----
All       All     []
`
	if diff := cmp.Diff(want, gotStr); diff != "" {
		t.Errorf("List output mismatch (-want +got):\n%s", diff)
	}
}
