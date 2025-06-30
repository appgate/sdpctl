package prompt

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/appgate/sdpctl/pkg/cmdutil"
)

var allowedSpecialChars = `!@#$%^&*()_+\-=\[\]{}|;':",./<>?~$`
var PassphraseInvalidMessage = fmt.Sprintf("Passphrase contains invalid characters. Only alphanumeric characters and the folowing special characters are permitted:%v", allowedSpecialChars)

func PasswordConfirmation(message string) (string, error) {
	firstAnswer, err := PromptPassword(message)
	if err != nil {
		return firstAnswer, err
	}
	secondAnswer, err := PromptPassword("Confirm your passphrase:")
	if err != nil {
		return firstAnswer, err
	}
	if firstAnswer != secondAnswer {
		return firstAnswer, errors.New("The passphrase did not match")
	}
	return firstAnswer, nil
}

// GetPassphrase check stdin if we have anything, and use that as passphrase
// otherwise, if we can prompt, Prompt user input
func GetPassphrase(stdIn io.Reader, canPrompt bool, hasStdin bool, message string) (string, error) {
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

// ValidateBackupPassphrase validates that a passphrase contains only allowed characters:
// alphanumeric characters (a-z, A-Z, 0-9) and common printable special characters.
// It rejects spaces, tabs, and exotic unicode characters like emojis.
func ValidateBackupPassphrase(passphrase string) error {
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}

	// Allow alphanumeric characters and common printable special characters
	// Explicitly exclude spaces, tabs, and unicode characters outside basic ASCII printable range
	allowedPattern := "^[a-zA-Z0-9" + allowedSpecialChars + "]+$"
	matched, err := regexp.MatchString(allowedPattern, passphrase)
	if err != nil {
		return fmt.Errorf("failed to validate passphrase: %w", err)
	}

	if !matched {
		return errors.New(PassphraseInvalidMessage)
	}

	return nil
}

// GetBackupPassphrase checks stdin if we have anything, and use that as passphrase
// otherwise, if we can prompt, prompt user input with validation for backup passphrases
func GetBackupPassphrase(stdIn io.Reader, canPrompt bool, hasStdin bool, message string) (string, error) {
	if hasStdin {
		buf, err := io.ReadAll(stdIn)
		if err != nil {
			return "", fmt.Errorf("could not read input from stdin %s", err)
		}
		passphrase := strings.TrimSuffix(string(buf), "\n")
		if err := ValidateBackupPassphrase(passphrase); err != nil {
			return "", err
		}
		return passphrase, nil
	}
	if !canPrompt {
		return "", cmdutil.ErrMissingTTY
	}
	return BackupPasswordConfirmation(message)
}

// BackupPasswordConfirmation prompts for a backup passphrase with validation
func BackupPasswordConfirmation(message string) (string, error) {
	for {
		firstAnswer, err := PromptPassword(message)
		if err != nil {
			return "", err
		}

		if err := ValidateBackupPassphrase(firstAnswer); err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			continue
		}

		secondAnswer, err := PromptPassword("Confirm your passphrase:")
		if err != nil {
			return "", err
		}

		if firstAnswer != secondAnswer {
			fmt.Println("Error: The passphrase did not match")
			continue
		}

		return firstAnswer, nil
	}
}
