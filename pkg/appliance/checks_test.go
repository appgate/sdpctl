package appliance

import (
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/go-cmp/cmp"
)

func TestShowDiskSpaceWarningMessage(t *testing.T) {
	type args struct {
		stats []openapi.StatsAppliancesListAllOfData
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "warning",
			args: args{
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Name: openapi.PtrString("controller"),
						Disk: openapi.PtrFloat32(90),
					},
				},
			},
			want: `
Some appliances have very little space available

  - controller  Disk usage: 90%


Upgrading requires the upload and decompression of big images.
To avoid problems during the upgrade process it's recommended to
increase the space on those appliances.
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ShowDiskSpaceWarningMessage(tt.args.stats)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShowDiskSpaceWarningMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", got, tt.want)
			}
		})
	}
}
