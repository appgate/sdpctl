package appliance

import (
	"regexp"
	"strings"

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
	versionRegex = regexp.MustCompile(`(\d+[.]\d+[.]\d+)-(\d+)-(\w+)`)
)

// GuessVersion tries to determine appliance version based on the input filename,
// It assumes the file is has the standard naming convention of
// appgate-5.4.4-26245-release.img.zip
// where 5.4.4 is the semver of the appliance.
func GuessVersion(input string) (*version.Version, error) {
	if versionRegex.MatchString(input) {
		edges := versionRegex.Split(input, 2)
		if len(edges) == 2 &&
			strings.HasPrefix(input, edges[0]) &&
			strings.HasSuffix(input, edges[1]) {
			v := strings.TrimSuffix(
				strings.TrimPrefix(input, edges[0]),
				edges[1],
			)
			// Correctly parse semver metadata with + instead of -
			return version.NewVersion(strings.Replace(v, "-", "+", 1))
		}
	}
	return version.NewVersion(input)
}
