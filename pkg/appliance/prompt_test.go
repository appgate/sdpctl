package appliance

import (
	"context"
	"testing"

	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
)

func TestPromptSelect(t *testing.T) {
	type args struct {
		filter     map[string]map[string]string
		orderBy    []string
		descending bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "select gateway",
			args: args{
				filter:  nil,
				orderBy: []string{"name"},
			},
			want:    "ee639d70-e075-4f01-596b-930d5f24f569",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			registry := httpmock.NewRegistry(t)
			a := &Appliance{
				APIClient: registry.Client,
				Token:     "",
			}
			registry.Register("/admin/appliances", httpmock.JSONResponse("../appliance/fixtures/appliance_list.json"))
			registry.Register("/admin/appliances/status", httpmock.JSONResponse("../appliance/fixtures/stats_appliance.json"))
			stubber, teardown := prompt.InitStubbers(t)
			func(s *prompt.PromptStubber) {
				s.StubOne(1)
			}(stubber)
			defer teardown()
			defer registry.Teardown()
			registry.Serve()
			got, err := PromptSelect(ctx, a, tt.args.filter, tt.args.orderBy, tt.args.descending)
			if (err != nil) != tt.wantErr {
				t.Errorf("PromptSelect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PromptSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}
