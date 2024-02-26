package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/cmdutil"
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
	signin(ctx context.Context, loginOpts openapi.LoginRequest, provider openapi.IdentityProvidersNamesGet200ResponseDataInner) (*signInResponse, error)
}

// mandatoryEnvVariables if no TTY is enable
var mandatoryEnvVariables = []string{
	"SDPCTL_USERNAME",
	"SDPCTL_PASSWORD",
}

func hasRequiredEnv() bool {
	for _, value := range mandatoryEnvVariables {
		if _, ok := os.LookupEnv(value); !ok {
			return false
		}
	}
	return true
}

// GetMinMaxAPIVersion sends a invalid authentication request
// to use the error response body to determine min, max supported version for the current api
//
// # Example response body
//
//	{
//	    "id": "not acceptable",
//	    "maxSupportedVersion": 17,
//	    "message": "Invalid 'Accept' header. Current version: application/vnd.appgate.peer-v17+json, Received: application/vnd.appgate.peer-v5+json",
//	    "minSupportedVersion": 13
//	}
func GetMinMaxAPIVersion(f *factory.Factory) (*MinMax, error) {
	cfg := f.Config
	client, err := f.APIClient(cfg)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	loginOpts := openapi.LoginRequest{
		DeviceId: cfg.DeviceID,
	}
	acceptHeaderFormatString := "application/vnd.appgate.peer-v%d+json"
	authenticator := NewAuth(client)
	// initial authentication, this will fail, since we will use the signin response
	// to compute the correct peerVersion used in the selected collective.
	_, minMax, err := authenticator.Authentication(context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf(acceptHeaderFormatString, 5)), loginOpts)
	if err != nil && minMax == nil {
		return nil, err
	}
	if minMax != nil {
		return minMax, nil
	}
	return nil, errors.New("Could not automatically determine api version to use")
}

var KeyringWarningMessage = "[warning] Could not integrate with system keyring. To disable keyring integration, set environment variable SDPCTL_NO_KEYRING=true"

var ErrSignInNotSupported = errors.New("No TTY present, and missing required environment variables to authenticate")

type authContext string

var contextKeyCanPrompt = authContext("canPrompt")

// Signin support interactive signin if a valid TTY is present, otherwise it requires environment variables to authenticate,
// this is only supported by 'local' auth provider
// If OTP is required, a prompt will appear and await user input
// Signin is done in several steps
// - Compute correct peer api version to use, based on login response body, which gives us a range of supported peer api to use
// - If there are more than 1 auth provider supported, prompt user to select (requires TTY | error shown if no TTY)
// - Store bearer token in os keyring, (refresh token if the provider supports it too)
// - Store the primary Controller version in config file
// - Save config file to $SDPCTL_CONFIG_DIR
func Signin(f *factory.Factory) error {
	cfg := f.Config
	client, err := f.APIClient(cfg)
	if err != nil {
		return err
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = configuration.DefaultDeviceID()
	}

	loginOpts := openapi.LoginRequest{
		DeviceId: cfg.DeviceID,
	}

	if cfg.Provider != nil {
		loginOpts.ProviderName = *cfg.Provider
	}

	authenticator := NewAuth(client)

	ctx := context.WithValue(context.Background(), contextKeyCanPrompt, f.CanPrompt())

	acceptHeaderFormatString := "application/vnd.appgate.peer-v%d+json"

	bearer, err := cfg.GetBearTokenHeaderValue()
	if err == nil && cfg.ExpiredAtValid() && len(bearer) > 0 && cfg.Version > 0 {
		// Make a test request and see if the locally stored auth bearer token
		// is valid, if we get any errors here, we can assume the token has been revoked.
		_, err := authenticator.Authorization(context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf(acceptHeaderFormatString, cfg.Version)), bearer)
		if err == nil {
			// if we don't get any errors here, we can be sure that the locally stored bearer token
			// is still valid.
			return nil
		}
	}

	minMax, err := GetMinMaxAPIVersion(f)
	if err != nil {
		return err
	}
	viper.Set("api_version", minMax.Max)
	cfg.Version = int(minMax.Max)

	acceptValue := fmt.Sprintf(acceptHeaderFormatString, minMax.Max)
	ctxWithAccept := context.WithValue(ctx, openapi.ContextAcceptHeader, acceptValue)
	providers, err := authenticator.ProviderNames(ctxWithAccept)
	if err != nil {
		return err
	}

	if len(providers) == 1 && len(loginOpts.ProviderName) == 0 {
		loginOpts.ProviderName = providers[0].GetName()
	}
	providerMap := make(map[string]openapi.IdentityProvidersNamesGet200ResponseDataInner, 0)
	providerNames := make([]string, 0)
	for _, p := range providers {
		providerMap[p.GetName()] = p
		providerNames = append(providerNames, p.GetName())
	}

	promptProvider := len(providers) > 1 && len(loginOpts.ProviderName) == 0

	if !f.CanPrompt() {
		if !hasRequiredEnv() {
			return ErrSignInNotSupported
		}
		if promptProvider {
			return fmt.Errorf("multiple providers available, but no TTY is present, set environment variable SDPCTL_PROVIDER")
		}
	}

	if promptProvider {
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
		return fmt.Errorf("invalid provider %s. Available providers: %s", loginOpts.ProviderName, strings.Join(providerNames, ", "))
	}
	cfg.Provider = &loginOpts.ProviderName
	var p Authenticate
	switch selectedProvider.GetType() {
	case RadiusProvider:
	case LocalProvider:
		p = NewLocal(f)
	case OidcProvider:
		if os.Getenv("SDPCTL_NO_KEYRING") != "" {
			return fmt.Errorf("%s provider does not work when environment variable SDPCTL_NO_KEYRING is set.", selectedProvider.GetType())
		}
		oidc := NewOpenIDConnect(f, client)
		defer oidc.Close()
		p = oidc
	default:
		return fmt.Errorf("%s %s identity provider is not supported", selectedProvider.GetName(), selectedProvider.GetType())
	}
	response, err := p.signin(ctxWithAccept, loginOpts, selectedProvider)
	if err != nil {
		return err
	}

	newToken, err := authAndOTP(ctxWithAccept, authenticator, response.LoginOpts.Password, response.Token)
	if err != nil {
		return err
	}

	authorizationToken, err := authenticator.Authorization(ctxWithAccept, *newToken)
	if err != nil {
		return err
	}
	cfg.BearerToken = openapi.PtrString(authorizationToken.GetToken())
	// use the original auth request expires_at value instead of the value from authorization since they can be different
	// depending on the provider type.
	cfg.ExpiresAt = openapi.PtrString(response.Expires.String())

	// if we have set environment variable SDPCTL_NO_KEYRING, we wont save anything to the keyring
	// default behavior is to save to keyring unless this environment variable is explicitly set.
	if os.Getenv("SDPCTL_NO_KEYRING") != "" {
		if err := os.Setenv("SDPCTL_BEARER", *cfg.BearerToken); err != nil {
			fmt.Fprintf(f.StdErr, "[warning] sdpctl keyring integration disabled by environment variable, cant set key to env %s\n", err)
			return nil
		}
		return nil
	}

	prefix, err := cfg.KeyringPrefix()
	if err != nil {
		return err
	}

	// if the bearer token can't be saved to the keychain, it will be exported as env variable
	// and saved in the config file as fallback, this should only happened if the system does not
	// support the keychain integration.
	if err := keyring.SetBearer(prefix, *cfg.BearerToken); err != nil {
		fmt.Fprintf(f.StdErr, "[warning] could not save token to keyring %s\n", err)
		fmt.Fprintln(f.StdErr, KeyringWarningMessage)
	}

	// store username and password if any in keyring, in practice only applicable on local provider
	if len(response.LoginOpts.GetUsername()) > 1 && len(response.LoginOpts.GetPassword()) > 1 {
		if err := cfg.StoreCredentials(response.LoginOpts.GetUsername(), response.LoginOpts.GetPassword()); err != nil {
			fmt.Fprintf(f.StdErr, "[warning] %s\n", err)
			fmt.Fprintln(f.StdErr, KeyringWarningMessage)
			return nil
		}
	}

	viper.Set("provider", selectedProvider.GetName())
	viper.Set("expires_at", cfg.ExpiresAt)
	viper.Set("url", cfg.URL)

	// If cert is entered, but not hashed in config, we'll do that here before returning as part deprecating the pem_filepath config key
	if cfg.PemBase64 == nil && len(cfg.PemFilePath) > 0 {
		cert, err := configuration.ReadPemFile(cfg.PemFilePath)
		if err != nil {
			return err
		}
		viper.Set("pem_base64", base64.StdEncoding.EncodeToString(cert.Raw))
		// If migration is done, remove pem_filepath value in config as to not confuse if the path leads to an old certificate
		viper.Set("pem_filepath", "")
	}

	// saving the config file is not a fatal error, we will only show a error message
	if err := viper.WriteConfig(); err != nil {
		fmt.Fprintf(f.StdErr, "[error] %s\n", err)
	}
	return nil
}

var ErrCantPromptOTP = errors.New("authentication requires one-time-password, but a TTY prompt is not allowed, can't continue")

// authAndOTP returns the authorized bearer header value and prompt user for OTP if its required
func authAndOTP(ctx context.Context, authenticator *Auth, password *string, token string) (*string, error) {
	authToken := fmt.Sprintf("Bearer %s", token)
	_, err := authenticator.Authorization(ctx, authToken)
	if errors.Is(err, ErrPreConditionFailed) {
		if v, ok := ctx.Value(contextKeyCanPrompt).(bool); ok && !v {
			return nil, ErrCantPromptOTP
		}
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
					if errors.Is(err, cmdutil.ErrExecutionCanceledByUser) {
						return nil, err
					}
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
