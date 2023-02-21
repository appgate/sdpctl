package files

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/stretchr/testify/assert"
)

func setupDeleteTest(t *testing.T) (*httpmock.Registry, *factory.Factory, *bytes.Buffer) {
	t.Helper()
	registry := httpmock.NewRegistry(t)

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

	return registry, f, stdout
}

func TestDeleteSingleFile(t *testing.T) {
	registry, f, out := setupDeleteTest(t)
	defer registry.Teardown()
	registry.Register("/files/appgate-6.0.1-29983-beta.img.zip", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	registry.Serve()

	cmd := NewFilesCmd(f)
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.PersistentFlags().Bool("descending", false, "")
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"delete", "appgate-6.0.1-29983-beta.img.zip"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	actual, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}

	expect := "appgate-6.0.1-29983-beta.img.zip: deleted\n"

	assert.Equal(t, expect, string(actual))
}

func TestDeleteAllFiles(t *testing.T) {
	registry, f, out := setupDeleteTest(t)
	defer registry.Teardown()
	registry.Register("/files", httpmock.JSONResponse("../../../pkg/appliance/fixtures/file_list.json"))
	registry.Register("/files/appgate-6.0.1-29983-beta.img.zip", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	registry.Register("/files/appgate-5.5.1-29983.img.zip", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	registry.Serve()

	cmd := NewFilesCmd(f)
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.PersistentFlags().Bool("descending", false, "")
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"delete", "--all"})

	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("executeC %s", err)
	}

	actual, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}

	expect := `appgate-5.5.1-29983.img.zip: deleted
appgate-6.0.1-29983-beta.img.zip: deleted
`

	assert.Equal(t, expect, string(actual))
}

func TestFilesDeleteNoInteractive(t *testing.T) {
	registry, f, out := setupDeleteTest(t)
	defer registry.Teardown()
	registry.Register("/files", httpmock.JSONResponse("../../../pkg/appliance/fixtures/file_list.json"))
	registry.Serve()

	cmd := NewFilesCmd(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "")
	cmd.PersistentFlags().StringSlice("order-by", []string{"name"}, "")
	cmd.PersistentFlags().Bool("descending", false, "")
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"delete", "--no-interactive"})

	_, err := cmd.ExecuteC()
	if err == nil {
		t.Fatalf("expected error, got no error")
	}

	assert.Equal(t, "No files were deleted", err.Error())
}
