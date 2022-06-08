package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/keyring"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/pkg/browser"
	"github.com/spf13/viper"
)

type signInResponse struct {
	Token     string
	Expires   time.Time
	LoginOpts *openapi.LoginRequest
}

type Authenticate interface {
	// signin should include context with correct Accept header and provider metadata
	// if successful, it should return the bearer token value and expiration date.
	signin(ctx context.Context, loginOpts openapi.LoginRequest, provider openapi.InlineResponse200Data) (*signInResponse, error)
}

// Signin is an interactive sign in function, that generates the config file
// Signin will show a interactive prompt to query the user for username, password and enter MFA if needed.
// and support SDPCTL_USERNAME & SDPCTL_PASSWORD environment variables.
// Signin supports MFA, compute a valid peer api version for selected appgate sdp collective.
func Signin(f *factory.Factory, remember bool) error {
	cfg := f.Config
	client, err := f.APIClient(cfg)
	if err != nil {
		return err
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = configuration.DefaultDeviceID()
	}

	// if we already have a valid bearer token, we will continue without
	// without any additional checks.
	if cfg.ExpiredAtValid() && len(cfg.BearerToken) > 0 && !remember {
		return nil
	}
	authenticator := NewAuth(client)
	// Get credentials from credentials file
	// Overwrite credentials with values set through environment variables

	loginOpts := openapi.LoginRequest{
		ProviderName: cfg.Provider,
		DeviceId:     cfg.DeviceID,
	}
	ctx := context.Background()
	acceptHeaderFormatString := "application/vnd.appgate.peer-v%d+json"
	// initial authtentication, this will fail, since we will use the singin response
	// to compute the correct peerVersion used in the selected appgate sdp collective.
	_, minMax, err := authenticator.Authentication(context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf(acceptHeaderFormatString, 5)), loginOpts)
	if err != nil && minMax == nil {
		return fmt.Errorf("invalid credentials %w", err)
	}
	if minMax != nil {
		viper.Set("api_version", minMax.Max)
		cfg.Version = int(minMax.Max)
	}

	acceptValue := fmt.Sprintf(acceptHeaderFormatString, minMax.Max)
	ctxWithAccept := context.WithValue(ctx, openapi.ContextAcceptHeader, acceptValue)
	providers, err := authenticator.ProviderNames(ctxWithAccept)
	if err != nil {
		return err
	}

	if len(providers) == 1 && len(loginOpts.ProviderName) <= 0 {
		loginOpts.ProviderName = providers[0].GetName()
	}
	providerMap := make(map[string]openapi.InlineResponse200Data, 0)
	providerNames := make([]string, 0)
	for _, p := range providers {
		providerMap[p.GetName()] = p
		providerNames = append(providerNames, p.GetName())
	}

	if len(providers) > 1 && len(loginOpts.ProviderName) <= 0 {
		qs := &survey.Select{
			Message: "Choose a provider:",
			Options: providerNames,
		}
		if err := prompt.SurveyAskOne(qs, &loginOpts.ProviderName); err != nil {
			return err
		}
	}
	selectedProvider, ok := providerMap[loginOpts.ProviderName]
	if !ok {
		return fmt.Errorf("invalid provider %s - %s", selectedProvider.GetName(), loginOpts.ProviderName)
	}
	cfg.Provider = loginOpts.ProviderName
	var token *signInResponse
	var p Authenticate
	switch selectedProvider.GetType() {
	case RadiusProvider:
	case LocalProvider:
		p = NewLocal(f, remember)
	default:
		return fmt.Errorf("%s %s identity provider is not supported", selectedProvider.GetName(), selectedProvider.GetType())
	}
	token, err = p.signin(ctxWithAccept, loginOpts, selectedProvider)
	if err != nil {
		return err
	}
	newToken, err := authAndOTP(ctxWithAccept, authenticator, token.LoginOpts.Password, token.Token)
	if err != nil {
		return err
	}

	authorizationToken, err := authenticator.Authorization(ctxWithAccept, *newToken)
	if err != nil {
		return err
	}
	cfg.BearerToken = authorizationToken.GetToken()
	// use the original auth request expires_at value instead of the value from authorization since they can be different
	// depending on the provider type.
	cfg.ExpiresAt = token.Expires.String()
	host, err := cfg.GetHost()
	if err != nil {
		return err
	}
	if err := keyring.SetBearer(host, cfg.BearerToken); err != nil {
		return fmt.Errorf("could not store token in keychain %w", err)
	}

	viper.Set("provider", selectedProvider.GetName())
	viper.Set("expires_at", cfg.ExpiresAt)
	viper.Set("url", cfg.URL)

	a, err := f.Appliance(cfg)
	if err != nil {
		return err
	}
	allAppliances, err := a.List(ctx, nil)
	if err != nil {
		return err
	}
	primaryController, err := appliancepkg.FindPrimaryController(allAppliances, host)
	if err != nil {
		return err
	}
	stats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	v, err := appliancepkg.GetApplianceVersion(*primaryController, stats)
	if err != nil {
		return err
	}
	viper.Set("primary_controller_version", v.String())
	if remember {
		if err := viper.WriteConfig(); err != nil {
			return err
		}
	}

	return nil
}

// authAndOTP returns the authorized bearer header value and prompt user for OTP if its required
func authAndOTP(ctx context.Context, authenticator *Auth, password *string, token string) (*string, error) {
	authToken := fmt.Sprintf("Bearer %s", token)
	_, err := authenticator.Authorization(ctx, authToken)
	if errors.Is(err, ErrPreConditionFailed) {
		otp, err := authenticator.InitializeOTP(ctx, password, authToken)
		if err != nil {
			return nil, err
		}
		testOTP := func() (*openapi.LoginResponse, error) {
			var answer string
			optKey := &survey.Password{
				Message: "Please enter your one-time password:",
			}
			if err := prompt.SurveyAskOne(optKey, &answer, survey.WithValidator(survey.Required)); err != nil {
				return nil, err
			}
			return authenticator.PushOTP(ctx, answer, authToken)
		}
		// TODO add support for RadiusChallenge, Push
		switch otpType := otp.GetType(); otpType {
		case "Secret":
			barcodeFile, err := BarcodeHTMLfile(otp.GetBarcode(), otp.GetSecret())
			if err != nil {
				return nil, err
			}
			fmt.Printf("\nOpen %s to scan the barcode to your authenticator app\n", barcodeFile.Name())
			fmt.Printf("\nIf you can’t use the barcode, enter %s in your authenticator app\n", otp.GetSecret())
			if err := browser.OpenURL(barcodeFile.Name()); err != nil {
				return nil, err
			}
			defer os.Remove(barcodeFile.Name())
			fallthrough

		case "AlreadySeeded":
			fallthrough
		default:
			// Give the user 3 attempts to enter the correct OTP key
			for i := 0; i < 3; i++ {
				newToken, err := testOTP()
				if err != nil {
					if errors.Is(err, ErrInvalidOneTimePassword) {
						fmt.Fprintf(os.Stderr, "[error] %s\n", err)
						continue
					}
				}
				if newToken != nil {
					t := fmt.Sprintf("Bearer %s", newToken.GetToken())
					return &t, nil
				}
			}
		}
	}
	return &authToken, err
}
