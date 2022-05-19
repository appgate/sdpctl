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

func IsTTY(out io.Writer) bool {
	if stdout, ok := out.(*os.File); ok {
		return isTerminal(stdout)
	}
	return false
}
