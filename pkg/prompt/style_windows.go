//go:build windows
// +build windows

package prompt

// SpinnerStyle for Windows has no special unicode characters, to support cmd.exe out-of-the-box.
var SpinnerStyle = []string{"-", "\\", "|", "/"}

// SpinnerDone intentionally left empty due to causing false positives in cmd.exe
var SpinnerDone = ""
