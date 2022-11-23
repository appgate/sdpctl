package prompt

import (
	"errors"

	"github.com/AlecAivazis/survey/v2"
)

func PasswordConfirmation(message string) (string, error) {
	var (
		firstAnswer, secondAnswer string
	)
	passwordPrompt := &survey.Password{
		Message: message,
	}
	if err := SurveyAskOne(passwordPrompt, &firstAnswer, survey.WithValidator(survey.Required)); err != nil {
		return firstAnswer, err
	}
	passwordPrompt.Message = "Confirm your passphrase:"
	if err := SurveyAskOne(passwordPrompt, &secondAnswer, survey.WithValidator(survey.Required)); err != nil {
		return firstAnswer, err
	}
	if firstAnswer != secondAnswer {
		return firstAnswer, errors.New("The passphrase did not match")
	}
	return firstAnswer, nil
}
