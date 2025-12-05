package serviceusers

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	supkg "github.com/appgate/sdpctl/pkg/serviceusers"
	"github.com/spf13/cobra"
)

type serviceUsersTestStub struct {
	desc           string
	args           []string
	askStubs       func(*prompt.PromptStubber)
	httpStubs      []httpmock.Stub
	wantOut        *regexp.Regexp
	wantExactMatch string
	wantErr        bool
	wantErrOut     *regexp.Regexp
}

func setupServiceUsersTest(t *testing.T, tC *serviceUsersTestStub) (*cobra.Command, *httpmock.Registry, *bytes.Buffer, func()) {
	registry := httpmock.NewRegistry(t)
	for _, v := range tC.httpStubs {
		registry.Register(v.URL, v.Responder)
	}
	registry.Serve()

	stdout := &bytes.Buffer{}
	stdin := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	in := io.NopCloser(stdin)

	f := &factory.Factory{
		Config: &configuration.Config{
			Debug:   false,
			URL:     fmt.Sprintf("http://localhost:%d", registry.Port),
			Version: 17,
		},
		IOOutWriter: stdout,
		Stdin:       in,
		StdErr:      stderr,
	}
	f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
		return registry.Client, nil
	}
	f.ServiceUsers = func(c *configuration.Config) (*supkg.ServiceUsersAPI, error) {
		apiClient, _ := f.APIClient(c)
		bearerToken, err := c.GetBearTokenHeaderValue()
		if err != nil {
			return nil, err
		}
		return supkg.NewServiceUsersAPI(apiClient.ServiceUsersApi, bearerToken), nil
	}

	cmd := NewServiceUsersCMD(f)
	cmd.PersistentFlags().Bool("no-interactive", false, "")
	cmd.SetArgs(tC.args)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	stubber, teardown := prompt.InitStubbers(t)

	if tC.askStubs != nil {
		tC.askStubs(stubber)
	}

	return cmd, registry, stdout, teardown
}
