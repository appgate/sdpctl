package appliance

import (
	"testing"

	"github.com/hashicorp/go-version"
)

var (
	ApplianceBigVersionConstraints, _ = version.NewConstraint(">= 606.335.1234")
	Appliance63Constraints, _         = version.NewConstraint(">= 6.3.10")
	Appliance54Constraints, _         = version.NewConstraint(">= 5.4.0-beta")
	Appliance55Constraints, _         = version.NewConstraint(">= 5.5.0-beta")
	Appliance50Constraints, _         = version.NewConstraint(">= 5.0.0-beta")
)

func TestParseVersionString(t *testing.T) {
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
			name: "Full file path",
			args: args{
				"/full/file/path/is/not/allowed/appgate-5.4-26245-release.img.zip",
			},
			constraints: Appliance54Constraints,
			wantErr:     false,
		},
		{
			name: "Swapped meta and pre",
			args: args{
				"appgate-5.4-beta-26245.img.zip",
			},
			constraints: Appliance54Constraints,
			wantErr:     false,
		},
		{
			name: "plus instead of minus",
			args: args{
				"appgate-5.4+release+26245.img.zip",
			},
			constraints: Appliance54Constraints,
			wantErr:     false,
		},
		{
			name: "major and minor version only",
			args: args{
				"5.4.img.zip",
			},
			constraints: Appliance54Constraints,
			wantErr:     false,
		},
		{
			name: "major version only",
			args: args{
				"5.img.zip",
			},
			constraints: Appliance50Constraints,
			wantErr:     false,
		},
		{
			name: "full version no file ending",
			args: args{
				"5.4.4",
			},
			constraints: Appliance54Constraints,
			wantErr:     false,
		},
		{
			name: "major and minor version only no file ending",
			args: args{
				"5.4",
			},
			constraints: Appliance54Constraints,
			wantErr:     false,
		},
		{
			name: "major version only no file ending",
			args: args{
				"5",
			},
			constraints: Appliance50Constraints,
			wantErr:     false,
		},
		{
			name: "empty string",
			args: args{
				"",
			},
			constraints: Appliance55Constraints,
			wantErr:     true,
		},
		{
			name: "full version with multidigit patch",
			args: args{
				"appgate-6.3.10+41534.img.zip",
			},
			constraints: Appliance63Constraints,
			wantErr:     false,
		},
		{
			name: "full version with multidigit patch",
			args: args{
				"appgate-606.335.1234+41534.img.zip",
			},
			constraints: ApplianceBigVersionConstraints,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersionString(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseVersionString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) && !tt.constraints.Check(got) {
				t.Errorf("%s does not satisfy constraints %s", got, tt.constraints)
			}
		})
	}
}
