package prompt

import (
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/appliance"
)

// AskConfirmation make sure user confirm action, otherwise abort.
func AskConfirmation(m ...string) error {
	m = append(m, "Do you want to continue?")
	ok := false
	prompt := &survey.Confirm{
		Message: strings.Join(m, "\n\n"),
	}
	if err := survey.AskOne(prompt, &ok); err != nil || !ok {
		return appliance.ErrExecutionCanceledByUser
	}
	return nil
}
