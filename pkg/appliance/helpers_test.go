package appliance

import (
	"testing"

	"github.com/hashicorp/go-version"
)

var (
	Appliance54Constraints, _ = version.NewConstraint(">= 5.4.0-*, < 5.5.0")
	Appliance55Constraints, _ = version.NewConstraint(">= 5.5.0-*, < 5.6.0")
)

func TestGuessVersion(t *testing.T) {
	type args struct {
		f string
	}

	tests := []struct {
		name        string
		args        args
		constraints version.Constraints
		wantErr     bool
	}{
		{
			name: "with metadata",
			args: args{
				"appgate-5.4.4-26245-release.img.zip",
			},
			constraints: Appliance54Constraints,
			wantErr:     false,
		},
		{
			name: "5.5 beta",
			args: args{
				"appgate-5.5.0-26245-beta.img.zip",
			},
			constraints: Appliance55Constraints,
			wantErr:     false,
		},
		{
			name: "Full file path not allowed",
			args: args{
				"/full/file/path/is/not/allowed/appgate-5.4-26245-release.img.zip",
			},
			constraints: Appliance55Constraints,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersionString(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GuessVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) && !tt.constraints.Check(got) {
				t.Errorf("%s does not satisfy constraints %s", got, tt.constraints)
			}
		})
	}
}
