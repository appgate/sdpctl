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

var lbsResponse = `
{
	"data": [
		"lbs1",
		"lbs2",
		"lbs3"

	],
	"totalCount": 3,
	"queries": [],
	"range": "0-3/3",
	"orderBy": "name",
	"descending": false,
	"filterBy": [],
	"resolver": "aws",
	"type": "lbs",
	"gatewayName": "gateway-site1"
}`

var folderResponse = `
{
	"data": [
		"folder1",
		"folder2",
		"folder3"

	],
	"totalCount": 3,
	"queries": [],
	"range": "0-3/3",
	"orderBy": "name",
	"descending": false,
	"filterBy": [],
	"resolver": "esx",
	"type": "folders",
	"gatewayName": "gateway-site1"
}`

var listVMResourceResponse = `
{
	"data": [
		"Windows10",
		"controller-endpoint",
		"controller-site1"

	],
	"totalCount": 3,
	"queries": [],
	"range": "0-3/3",
	"orderBy": "name",
	"descending": false,
	"filterBy": [],
	"resolver": "esx",
	"type": "virtual-machines",
	"gatewayName": "gateway-site1"
}`

var emptyResponse = `
{
	"data": [
	],
	"totalCount": 0,
	"queries": [],
	"range": "0-0/0",
	"orderBy": "name",
	"descending": false,
	"filterBy": [],
	"resolver": "esx",
	"type": "virtual-machines",
	"gatewayName": "gateway-site1"
}`

var statusStubs = []httpmock.Stub{
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
			}

var resourceStubs = []httpmock.Stub{
				statusStubs[0],
				{
					URL: "/admin/sites/1e478198-fa88-4368-a4ee-4c06e0de0744/resources",
					
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						
						r.ParseForm()
						if r.Form.Get("resolver") == "esx" && r.Form.Get("type")== "virtual-machines"{
							w.Write([]byte(listVMResourceResponse))
						} else if r.Form.Get("resolver") == "esx" && r.Form.Get("type")== "folders"{
							w.Write([]byte(folderResponse))
						}else if r.Form.Get("resolver") == "aws" && r.Form.Get("type")== "lbs"{
							w.Write([]byte(lbsResponse))
						}else{
							w.Write([]byte(emptyResponse))
						}
						

						
					},
				},
			}

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
			stubs: statusStubs,
			want: `Name    Resolver    Type    Gateway Name
----    --------    ----    ------------
No resources found in the site`,
		},
{
			desc: "list resources with all returned",
			cli:  "resources 1e478198-fa88-4368-a4ee-4c06e0de0744",
			stubs: resourceStubs,
			want: `Name                   Resolver    Type                Gateway Name
----                   --------    ----                ------------
lbs1                   aws         lbs                 gateway-site1
lbs2                   aws         lbs                 gateway-site1
lbs3                   aws         lbs                 gateway-site1
Windows10              esx         virtual-machines    gateway-site1
controller-endpoint    esx         virtual-machines    gateway-site1
controller-site1       esx         virtual-machines    gateway-site1
folder1                esx         folders             gateway-site1
folder2                esx         folders             gateway-site1
folder3                esx         folders             gateway-site1`,
		},
{
			desc: "list resources with filtering",
			cli:  `resources 1e478198-fa88-4368-a4ee-4c06e0de0744 --resolver "esx" --resource "folders"`,
			stubs: resourceStubs,
			want: `Name       Resolver    Type       Gateway Name
----       --------    ----       ------------
folder1    esx         folders    gateway-site1
folder2    esx         folders    gateway-site1
folder3    esx         folders    gateway-site1`,
		},
{
			desc: "list resources with multiple filtering",
			cli:  `resources 1e478198-fa88-4368-a4ee-4c06e0de0744 --resolver "esx&aws" --resource "folders&lbs"`,
			stubs: resourceStubs,
			want: `Name       Resolver    Type       Gateway Name
----       --------    ----       ------------
folder1    esx         folders    gateway-site1
folder2    esx         folders    gateway-site1
folder3    esx         folders    gateway-site1
lbs1       aws         lbs        gateway-site1
lbs2       aws         lbs        gateway-site1
lbs3       aws         lbs        gateway-site1`,
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
