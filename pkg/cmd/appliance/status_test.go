package appliance

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/cmd/factory"
	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/go-cmp/cmp"
)

type CmdOut struct {
	OutBuf *bytes.Buffer
	ErrBuf *bytes.Buffer
}

func TestUpgradeStatusCommand(t *testing.T) {
	registery := httpmock.NewRegistry()
	registery.Register(
		"/appliances",
		httpmock.FileResponse("../../appliance/fixtures/applianceList.json"),
	)
	registery.Register(
		"/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
		httpmock.FileResponse("../../appliance/fixtures/appliance_upgrade_status_idle.json"),
	)
	defer registery.Teardown()
	registery.Serve()
	stdout := &bytes.Buffer{}
	f := &factory.Factory{
		Config: &config.Config{
			Debug: false,
			URL:   fmt.Sprintf("http://localhost:%d", registery.Port),
		},
		IOOutWriter: stdout,
		APIClient: func(c *config.Config) (*openapi.APIClient, error) {
			return registery.Client, nil
		},
	}

	cmd := NewUpgradeStatusCmd(f)

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

	want := []byte(`[
	    {
	      "id": "4c07bc67-57ea-42dd-b702-c2d6c45419fc",
	      "name": "object",
	      "status": "idle",
	      "details": "a reboot is required for the Upgrade to go into effect"
	    }
	]`)

	if diff := cmp.Diff(want, got, httpmock.TransformJSONFilter); diff != "" {
		t.Fatalf("JSON Diff (-want +got):\n%s", diff)
	}
}
