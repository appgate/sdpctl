package prompt

import (
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/appgate/appgatectl/pkg/cmdutil"
)

// AskConfirmation make sure user confirm action, otherwise abort.
func AskConfirmation(m ...string) error {
	m = append(m, "Do you want to continue?")
	ok := false
	p := &survey.Confirm{
		Message: strings.Join(m, "\n\n"),
	}
	if err := SurveyAskOne(p, &ok); err != nil || !ok {
		return cmdutil.ErrExecutionCanceledByUser
	}
	return nil
}

// SurveyAskOne helper method, mainly used within tests
var SurveyAskOne = func(p survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
	err := survey.AskOne(p, response, opts...)
	if err != nil {
		if err == terminal.InterruptErr {
			return cmdutil.ErrExecutionCanceledByUser
		}
		return err
	}
	return nil
}
