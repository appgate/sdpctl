package appliance

import (
	"context"
	"testing"

	"github.com/appgate/appgatectl/pkg/httpmock"
	"github.com/appgate/appgatectl/pkg/prompt"
)

func TestPromptSelect(t *testing.T) {
	type args struct {
		filter map[string]map[string]string
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
				filter: nil,
			},
			want:    "ee639d70-e075-4f01-596b-930d5f24f569",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			registry := httpmock.NewRegistry()
			a := &Appliance{
				APIClient: registry.Client,
				Token:     "",
			}
			registry.Register("/appliances", httpmock.JSONResponse("../appliance/fixtures/appliance_list.json"))
			registry.Register("/stats/appliances", httpmock.JSONResponse("../appliance/fixtures/stats_appliance.json"))
			stubber, teardown := prompt.InitAskStubber()
			func(s *prompt.AskStubber) {
				s.StubOne(1)
			}(stubber)
			defer teardown()
			defer registry.Teardown()
			registry.Serve()
			got, err := PromptSelect(ctx, a, tt.args.filter)
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
