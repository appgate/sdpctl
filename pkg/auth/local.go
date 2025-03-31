package auth

import (
	"context"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
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
		credentials.Username, err = prompt.PromptInput("Username:")
		if err != nil {
			return nil, err
		}
	}
	if len(credentials.Password) <= 0 && canPrompt {
		credentials.Password, err = prompt.PromptPassword("Password:")
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
