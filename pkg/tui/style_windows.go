//go:build windows
// +build windows

package tui

var (
	// SpinnerStyle for Windows has no special unicode characters, to support cmd.exe out-of-the-box.
	SpinnerStyle []string = []string{"-", "\\", "|", "/"}

	Check string = "[COMPLETE]"
	Cross string = "[ERROR]"
	Yes   string = "Y"
	No    string = "N"
)
