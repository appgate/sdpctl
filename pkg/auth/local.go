package auth

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
)

type Local struct {
	Factory  *factory.Factory
	Remember bool
}

func NewLocal(f *factory.Factory, remember bool) *Local {
	return &Local{
		Factory:  f,
		Remember: remember,
	}
}

func (l Local) signin(ctx context.Context, loginOpts openapi.LoginRequest, provider openapi.InlineResponse200Data) (*signInResponse, error) {
	cfg := l.Factory.Config

	// Clear old credentials if remember me flag is provided
	if l.Remember {
		if err := cfg.ClearCredentials(); err != nil {
			return nil, err
		}
	}

	client, err := l.Factory.APIClient(cfg)
	if err != nil {
		return nil, err
	}
	authenticator := NewAuth(client)
	credentials, err := cfg.LoadCredentials()
	if err != nil {
		return nil, err
	}

	if len(credentials.Username) <= 0 {
		err := prompt.SurveyAskOne(&survey.Input{
			Message: "Username:",
		}, &credentials.Username, survey.WithValidator(survey.Required))
		if err != nil {
			return nil, err
		}
	}
	if len(credentials.Password) <= 0 {
		err := prompt.SurveyAskOne(&survey.Password{
			Message: "Password:",
		}, &credentials.Password, survey.WithValidator(survey.Required))
		if err != nil {
			return nil, err
		}
	}

	if l.Remember {
		if err := rememberCredentials(cfg, credentials); err != nil {
			return nil, fmt.Errorf("Failed to store credentials: %w", err)
		}
	}
	loginOpts.Username = openapi.PtrString(credentials.Username)
	loginOpts.Password = openapi.PtrString(credentials.Password)

	loginResponse, _, err := authenticator.Authentication(ctx, loginOpts)
	if err != nil {
		return nil, err
	}
	response := &signInResponse{
		Token:     loginResponse.GetToken(),
		Expires:   loginResponse.GetExpires(),
		LoginOpts: &loginOpts,
	}
	return response, nil
}

func rememberCredentials(cfg *configuration.Config, credentials *configuration.Credentials) error {
	q := []*survey.Question{
		{
			Name: "remember",
			Prompt: &survey.Select{
				Message: "What credentials should be saved?",
				Options: []string{"both", "only username", "only password"},
				Default: "both",
			},
		},
	}

	answers := struct {
		Remember string `survey:"remember"`
	}{}

	if err := survey.Ask(q, &answers); err != nil {
		return err
	}

	credentialsCopy := &configuration.Credentials{}
	switch answers.Remember {
	case "only username":
		credentialsCopy.Username = credentials.Username
	case "only password":
		credentialsCopy.Password = credentials.Password
	default:
		credentialsCopy.Username = credentials.Username
		credentialsCopy.Password = credentials.Password
	}

	if err := cfg.StoreCredentials(credentialsCopy); err != nil {
		return err
	}

	return nil
}
