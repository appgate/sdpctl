package sites

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestSitesListCommand(t *testing.T) {
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
			cli:     "list",
			wantErr: true,
		},
		{
			desc: "list no configured site test",
			cli:  "list",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Write([]byte{})
					},
				},
			},
			want: `No sites configured in the collective`,
		},
		{
			desc: "no site matching argument",
			cli:  "list SomeSiteName",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := openapi.SiteWithStatusList{
							Data: []openapi.SiteWithStatus{
								{
									Name: "Not This Site",
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
			want: "No sites available matching the arguments",
		},
		{
			desc: "one site matching argument",
			cli:  "list SomeSiteName",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := openapi.SiteWithStatusList{
							Data: []openapi.SiteWithStatus{
								{
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
			want: `Site Name       Short Name    ID    Tags    Description    Status
---------       ----------    --    ----    -----------    ------
SomeSiteName                        []`,
		},
		{
			desc: "one site matching partial argument",
			cli:  "list SiteName",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := openapi.SiteWithStatusList{
							Data: []openapi.SiteWithStatus{
								{
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
			want: `Site Name       Short Name    ID    Tags    Description    Status
---------       ----------    --    ----    -----------    ------
SomeSiteName                        []`,
		},
		{
			desc: "one site matching uuid argument",
			cli:  "list 32bf476b-0ab9-4d9d-879e-321651586b6a",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := openapi.SiteWithStatusList{
							Data: []openapi.SiteWithStatus{
								{
									Id:   openapi.PtrString("32bf476b-0ab9-4d9d-879e-321651586b6a"),
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
			want: `Site Name       Short Name    ID                                      Tags    Description    Status
---------       ----------    --                                      ----    -----------    ------
SomeSiteName                  32bf476b-0ab9-4d9d-879e-321651586b6a    []`,
		},
		{
			desc: "list sites no argument",
			cli:  "list",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := openapi.SiteWithStatusList{
							Data: []openapi.SiteWithStatus{
								{
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
			want: `Site Name       Short Name    ID    Tags    Description    Status
---------       ----------    --    ----    -----------    ------
SomeSiteName                        []`,
		},
		{
			desc: "list sites with multiline description",
			cli:  "list",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/sites/status",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := openapi.SiteWithStatusList{
							Data: []openapi.SiteWithStatus{
								{
									Name:        "SomeSiteName",
									Description: openapi.PtrString("This is a comment\nspanning multiple lines"),
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
			want: `Site Name       Short Name    ID    Tags    Description                Status
---------       ----------    --    ----    -----------                ------
SomeSiteName                        []      This is a comment [...]`,
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
