package auth

import (
	"context"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
)

type Local struct {
	Factory *factory.Factory
}

func NewLocal(f *factory.Factory) *Local {
	return &Local{
		Factory: f,
	}
}

func (l Local) signin(ctx context.Context, loginOpts openapi.LoginRequest, provider openapi.IdentityProvidersNamesGet200ResponseDataInner) (*signInResponse, error) {
	cfg := l.Factory.Config
	canPrompt := l.Factory.CanPrompt()
	client, err := l.Factory.APIClient(cfg)
	if err != nil {
		return nil, err
	}
	authenticator := NewAuth(client)
	credentials, err := cfg.LoadCredentials()
	if err != nil {
		return nil, err
	}

	if len(credentials.Username) <= 0 && canPrompt {
		err := prompt.SurveyAskOne(&survey.Input{
			Message: "Username:",
		}, &credentials.Username, survey.WithValidator(survey.Required))
		if err != nil {
			return nil, err
		}
	}
	if len(credentials.Password) <= 0 && canPrompt {
		err := prompt.SurveyAskOne(&survey.Password{
			Message: "Password:",
		}, &credentials.Password, survey.WithValidator(survey.Required))
		if err != nil {
			return nil, err
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
