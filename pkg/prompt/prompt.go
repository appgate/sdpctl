package prompt

import (
	"strings"

	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/tui"
)

// AskConfirmation make sure user confirm action, otherwise return error.
func AskConfirmation(m ...string) error {
	ok := tui.YesNo(strings.Join(m, "\n\n"), false)
	if !ok {
		return cmdutil.ErrExecutionCanceledByUser
	}
	return nil
}
