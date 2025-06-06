package sites

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)


func TestResourcesCommand(t *testing.T) {
	testCases := []struct {
		desc    string
		cli     string
		version string
		stubs   []httpmock.Stub
		want    string
		wantErr bool
	}{
		{
			desc:    "basic failing test",
			cli:     "resources",
			wantErr: true,
		},
{
			desc: "list resources none available",
			cli:  "resources 1e478198-fa88-4368-a4ee-4c06e0de0744",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := openapi.SiteWithStatusList{
							Data: []openapi.SiteWithStatus{
								{
									Id:   openapi.PtrString("1e478198-fa88-4368-a4ee-4c06e0de0744"),
									Name: "SomeSiteName",
								},
							},
						}
						b, err := json.Marshal(res)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.Header().Add("Content-Type", "application/json")
						w.Write(b)
					},
				},
			},
			want: `Name    Resolver    Type    Gateway Name
----    --------    ----    ------------
No resources found in the site`,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// mock server
			registry := httpmock.NewRegistry(t)
			for _, m := range tC.stubs {
				registry.Register(m.URL, m.Responder)
			}
			defer registry.Teardown()
			registry.Serve()

			buf := new(bytes.Buffer)
			f := &factory.Factory{
				Config: &configuration.Config{
					Debug: false,
					URL:   fmt.Sprintf("http://appgate.test:%d", registry.Port),
				},
				IOOutWriter: buf,
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registry.Client, nil
			}
			f.BaseURL = func() string { return f.Config.URL }

			// command
			cmd := NewSitesCmd(f)
			args, err := shlex.Split(tC.cli)
			if err != nil {
				t.Fatal(err)
			}
			cmd.SetArgs(args)
			_, err = cmd.ExecuteC()
			if tC.wantErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tC.want)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tC.want, strings.TrimSpace(buf.String()))
		})
}
}
