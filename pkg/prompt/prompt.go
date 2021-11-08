package prompt

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/appliance"
)

var SurveyAsk = func(qs []*survey.Question, response interface{}, opts ...survey.AskOpt) error {
	return survey.Ask(qs, response, opts...)
}

// AskConfirmation make sure user confirm action, otherwise abort.
func AskConfirmation() error {
	ok := false
	prompt := &survey.Confirm{
		Message: "Do you want to continue?",
	}
	if err := survey.AskOne(prompt, &ok); err != nil || !ok {
		return appliance.ErrExecutionCanceledByUser
	}
	return nil
}
