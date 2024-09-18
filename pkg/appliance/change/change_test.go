package change

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/go-cmp/cmp"
)

func TestApplianceChangeGet(t *testing.T) {
	type args struct {
		changeID    string
		applianceID string
	}
	tests := []struct {
		name      string
		httpStubs []httpmock.Stub
		args      args
		want      *openapi.AppliancesIdChangeChangeIdGet200Response
		wantErr   bool
	}{
		{
			name: "change running HTTP 200",
			httpStubs: []httpmock.Stub{
				{
					URL: "/admin/appliances/de0d2354-13cf-4c9e-8044-80563e340764/change/96525303-ef06-4f9b-922c-4a940e5b505e",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{
                            "details": "Disabling the Controller maintenance mode",
                            "id": "96525303-ef06-4f9b-922c-4a940e5b505e",
                            "status": "running"
                        }`))
					},
				},
			},
			args: args{
				changeID:    "96525303-ef06-4f9b-922c-4a940e5b505e",
				applianceID: "de0d2354-13cf-4c9e-8044-80563e340764",
			},
			want: &openapi.AppliancesIdChangeChangeIdGet200Response{
				Details: openapi.PtrString("Disabling the Controller maintenance mode"),
				Id:      "96525303-ef06-4f9b-922c-4a940e5b505e",
				Status:  "running",
			},
			wantErr: false,
		},
		{
			name: "change error http 500",
			httpStubs: []httpmock.Stub{
				{
					URL: "/admin/appliances/de0d2354-13cf-4c9e-8044-80563e340764/change/96525303-ef06-4f9b-922c-4a940e5b505e",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusServiceUnavailable)
					},
				},
			},
			args: args{
				changeID:    "96525303-ef06-4f9b-922c-4a940e5b505e",
				applianceID: "de0d2354-13cf-4c9e-8044-80563e340764",
			},

			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := httpmock.NewRegistry(t)
			for _, v := range tt.httpStubs {
				registry.Register(v.URL, v.Responder)
			}
			defer registry.Teardown()
			registry.Serve()

			ac := &ApplianceChange{
				APIClient: registry.Client,
				Token:     "tt.fields.Token",
			}
			got, err := ac.Get(context.TODO(), tt.args.changeID, tt.args.applianceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplianceChange.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("diff: %+v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestApplianceChangeRetryUntilCompleted(t *testing.T) {
	type args struct {
		changeID    string
		applianceID string
	}
	tests := []struct {
		name      string
		httpStubs []httpmock.Stub
		args      args
		want      *openapi.AppliancesIdChangeChangeIdGet200Response
		wantErr   bool
	}{
		{
			name: "change running HTTP 200",
			httpStubs: []httpmock.Stub{
				{
					URL: "/admin/appliances/de0d2354-13cf-4c9e-8044-80563e340764/change/96525303-ef06-4f9b-922c-4a940e5b505e",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{
                            "id": "96525303-ef06-4f9b-922c-4a940e5b505e",
                            "result": "success",
                            "status": "completed"
                        }`))
					},
				},
			},
			args: args{
				changeID:    "96525303-ef06-4f9b-922c-4a940e5b505e",
				applianceID: "de0d2354-13cf-4c9e-8044-80563e340764",
			},
			want: &openapi.AppliancesIdChangeChangeIdGet200Response{
				Id:     "96525303-ef06-4f9b-922c-4a940e5b505e",
				Status: "completed",
				Result: openapi.PtrString("success"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := httpmock.NewRegistry(t)
			for _, v := range tt.httpStubs {
				registry.Register(v.URL, v.Responder)
			}
			defer registry.Teardown()
			registry.Serve()

			ac := &ApplianceChange{
				APIClient: registry.Client,
				Token:     "tt.fields.Token",
			}
			got, err := ac.RetryUntilCompleted(context.TODO(), tt.args.changeID, tt.args.applianceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplianceChange.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("diff: %+v", cmp.Diff(got, tt.want))
			}
		})
	}
}
