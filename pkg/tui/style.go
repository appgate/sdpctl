//go:build !windows
// +build !windows

package tui

var (
	SpinnerStyle []string = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	Check string = "✓"
	Cross string = "⨯"
	Yes   string = Check
	No    string = Cross
)
