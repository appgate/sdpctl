package appliance

import (
	"fmt"
	"regexp"
	"strconv"

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
	versionRegex = regexp.MustCompile(`(([\d][.]?){1,3})[-|+]?([\d|\w]+)?[-|+]?([\d|\w]+)?(\.img\.zip)?$`)
)

// ParseVersionString tries to determine appliance version based on the input filename,
// It assumes the file is has the standard naming convention of
// appgate-5.4.4-26245-release.img.zip
// where 5.4.4 is the semver of the appliance.
func ParseVersionString(input string) (*version.Version, error) {
	m := versionRegex.FindStringSubmatch(input)
	var pre string
	var meta string
	if len(m) > 0 {
		input = m[1]
		if _, err := strconv.ParseInt(m[3], 10, 64); err == nil {
			meta = m[3]
			if len(m[4]) > 0 {
				pre = m[4]
			}
		}
		if _, err := strconv.ParseInt(m[4], 10, 64); err == nil {
			meta = m[4]
			if len(m[3]) > 0 {
				pre = m[3]
			}
		}
		if len(pre) > 0 && pre != "release" {
			input = fmt.Sprintf("%s-%s", input, pre)
		}
		if len(meta) > 0 {
			input = fmt.Sprintf("%s+%s", input, meta)
		}
	}
	return version.NewVersion(input)
}
