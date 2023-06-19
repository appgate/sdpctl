package appliance

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/hashicorp/go-version"
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
	v, err := version.NewVersion(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version string '%s': %w", input, err)
	}
	return v, nil
}

func ParseVersionFromZip(filename string) (*version.Version, error) {
	type metadata struct {
		Version string
	}
	zf, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zf.Close()

	for _, file := range zf.File {
		if file.Name == "metadata.json" {
			fd, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer fd.Close()

			content, err := io.ReadAll(fd)
			if err != nil {
				return nil, err
			}

			meta := metadata{}
			if err := json.Unmarshal(content, &meta); err != nil {
				return nil, err
			}
			return ParseVersionString(meta.Version)
		}
	}
	return nil, errors.New("no version found")
}
