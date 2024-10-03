package prompt

import (
	"fmt"
	"io"
	"strings"

	"github.com/appgate/sdpctl/pkg/tui"
)


// GetPassphrase check stdin if we have anything, and use that as passphrase
// otherwise, if we can prompt, Prompt user input
func GetPassphrase(stdIn io.Reader, canPrompt, hasStdin bool, message string) (string, error) {
	if hasStdin {
		buf, err := io.ReadAll(stdIn)
		if err != nil {
			return "", fmt.Errorf("could not read input from stdin %s", err)
		}
		return strings.TrimSuffix(string(buf), "\n"), nil
	}
	if !canPrompt {
		return "", fmt.Errorf("could not read input from tty")
	}
	return tui.Password(message)
}
