package filesystem

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"

	"github.com/adrg/xdg"
	"github.com/sirupsen/logrus"
)

const (
	AgConfigDir   = "SDPCTL_CONFIG_DIR"
	XdgConfigHome = "XDG_CONFIG_HOME"
	AppData       = "AppData"
)

func ConfigDir() string {
	if path := os.Getenv(AgConfigDir); len(path) > 0 {
		return path
	}
	if path := os.Getenv(XdgConfigHome); len(path) > 0 {
		return filepath.Join(path, "sdpctl")
	}
	return filepath.Join(xdg.ConfigHome, "sdpctl")
}

func DownloadDir() string {
	// xdg library does not currently parse the user-dirs.dirs file (see https://github.com/adrg/xdg/issues/29)
	// we'll do it manually for now
	ud, err := parseUsersDirs()
	if err != nil {
		// Not a fatal error here, since we have some fallback below
		logrus.WithError(err).Error("failed to read localized user directories from user-dirs.dirs")
	}
	if dlDir, ok := ud["DOWNLOAD"]; ok {
		return dlDir
	}
	return xdg.UserDirs.Download
}

func parseUsersDirs() (map[string]string, error) {
	res := map[string]string{}
	file, err := os.Open(filepath.Join(xdg.ConfigHome, "user-dirs.dirs"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	regex := regexp.MustCompile(`^XDG_(.+)_DIR="(.*)"$`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		if m := regex.FindStringSubmatch(txt); len(m) > 0 {
			res[m[1]] = os.ExpandEnv(m[2])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
