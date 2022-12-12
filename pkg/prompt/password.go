package prompt

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/cmdutil"
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
		return "", cmdutil.ErrMissingTTY
	}
	return PasswordConfirmation(message)
}
