package cmdutil

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

var isTerminal = func(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || IsCygwinTerminal(f)
}

func IsCygwinTerminal(f *os.File) bool {
	return isatty.IsCygwinTerminal(f.Fd())
}

func IsCI() bool {
	return os.Getenv("CI") != "" || // GitHub Actions, Travis CI, CircleCI, Cirrus CI, GitLab CI, AppVeyor, CodeShip, dsari
		os.Getenv("BUILD_NUMBER") != "" || // Jenkins, TeamCity
		os.Getenv("RUN_ID") != "" // TaskCluster, dsari
}

func IsTTY(out io.Writer) bool {
	if stdout, ok := out.(*os.File); ok {
		return isTerminal(stdout)
	}
	return false
}
