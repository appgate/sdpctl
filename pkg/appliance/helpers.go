package appliance

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/go-version"
)

const (
	HelpManualURL = "https://sdphelp.appgate.com/adminguide/v5.5"

	BackupInstructions = `
Please perform a backup or snapshot of %s before continuing!
Use appgate-backup to perform a backup of the Controller.
For more documentation on the backup process, go to:
    %s/backup-script.html
`
)

var (
	versionRegex = regexp.MustCompile(`(\d+[.]\d+[.]\d+)-?(\d+)?-?(\w+)?`)
)

// ParseVersionString tries to determine appliance version based on the input filename,
// It assumes the file is has the standard naming convention of
// appgate-5.4.4-26245-release.img.zip
// where 5.4.4 is the semver of the appliance.
func ParseVersionString(input string) (*version.Version, error) {
	m := versionRegex.FindStringSubmatch(input)
	if len(m) > 0 {
		input = m[1]
		if len(m[3]) > 0 && m[3] != "release" {
			input = fmt.Sprintf("%s-%s", input, m[3])
		}
		if len(m[2]) > 0 {
			input = fmt.Sprintf("%s+%s", input, m[2])
		}
	}
	return version.NewVersion(input)
}
