package entitlements

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNamesMigrationCommand(t *testing.T) {
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
			cli:     "names-migration",
			wantErr: true,
			stubs:   []httpmock.Stub{},
		},
		{
			desc: "no migrations needed",
			cli:  "names-migration",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/entitlements/cloud-migrations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := struct {
							Data []interface{} `json:"data"`
						}{
							Data: []interface{}{},
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
			want: `Name    ID    Original Value    Updated Value
----    --    --------------    -------------`,
		},
		{
			desc: "single migration",
			cli:  "names-migration",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/entitlements/cloud-migrations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := struct {
							Data []map[string]interface{} `json:"data"`
						}{
							Data: []map[string]interface{}{
								{
									"entitlementName": "Test Entitlement",
									"entitlementId":   "123e4567-e89b-12d3-a456-426614174000",
									"originalHost":    "old-hostname.example.com",
									"updatedHost":     "new-hostname.example.com",
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
			want: `Name                ID                                      Original Value              Updated Value
----                --                                      --------------              -------------
Test Entitlement    123e4567-e89b-12d3-a456-426614174000    old-hostname.example.com    new-hostname.example.com`,
		},
		{
			desc: "multiple migrations",
			cli:  "names-migration",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/entitlements/cloud-migrations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := struct {
							Data []map[string]interface{} `json:"data"`
						}{
							Data: []map[string]interface{}{
								{
									"entitlementName": "First Entitlement",
									"entitlementId":   "123e4567-e89b-12d3-a456-426614174000",
									"originalHost":    "old-host1.example.com",
									"updatedHost":     "new-host1.example.com",
								},
								{
									"entitlementName": "Second Entitlement",
									"entitlementId":   "223e4567-e89b-12d3-a456-426614174001",
									"originalHost":    "old-host2.example.com",
									"updatedHost":     "new-host2.example.com",
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
			want: `Name                  ID                                      Original Value           Updated Value
----                  --                                      --------------           -------------
First Entitlement     123e4567-e89b-12d3-a456-426614174000    old-host1.example.com    new-host1.example.com
Second Entitlement    223e4567-e89b-12d3-a456-426614174001    old-host2.example.com    new-host2.example.com`,
		},
		{
			desc: "migration with null updated host",
			cli:  "names-migration",
			stubs: []httpmock.Stub{
				{
					URL: "/admin/entitlements/cloud-migrations",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						res := struct {
							Data []map[string]interface{} `json:"data"`
						}{
							Data: []map[string]interface{}{
								{
									"entitlementName": "Test Entitlement",
									"entitlementId":   "123e4567-e89b-12d3-a456-426614174000",
									"originalHost":    "old-hostname.example.com",
									"updatedHost":     nil,
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
			want: `Name                ID                                      Original Value              Updated Value
----                --                                      --------------              -------------
Test Entitlement    123e4567-e89b-12d3-a456-426614174000    old-hostname.example.com`,
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
			cmd := NewEntitlementsMigrationCmd(f)
			args, err := shlex.Split(tC.cli)
			if err != nil {
				t.Fatal(err)
			}
			cmd.SetArgs(args)
			_, err = cmd.ExecuteC()
			if tC.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tC.want, strings.TrimSpace(buf.String()))
		})
	}
}
