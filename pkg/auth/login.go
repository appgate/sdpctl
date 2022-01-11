package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/cli/browser"
	"github.com/spf13/viper"
)

// Login is an interactive login function, that generates the config file
// Login will show a interactive prompt to query the user for username, password and enter MFA if needed.
// and support APPGATECTL_USERNAME & APPGATECTL_PASSWORD environment variables.
// Login supports MFA, compute a valid peer api version for selected appgate sdp collective.
func Login(f *factory.Factory, remember, saveConfig bool) error {
	cfg := f.Config
	client, err := f.APIClient(cfg)
	if err != nil {
		return err
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = configuration.DefaultDeviceID()
	}
	host, err := cfg.GetHost()
	if err != nil {
		return err
	}

	authenticator := NewAuth(client)
	// Get credentials from credentials file
	// Overwrite credentials with values set through environment variables
	credentials, err := cfg.LoadCredentials()
	if err != nil {
		return err
	}
	if envUsername := util.Getenv("APPGATECTL_USERNAME", viper.GetString("username")); len(envUsername) > 0 {
		credentials.Username = envUsername
	}
	if envPassword := util.Getenv("APPGATECTL_PASSWORD", viper.GetString("password")); len(envPassword) > 0 {
		credentials.Password = envPassword
	}

	if len(credentials.Username) <= 0 {
		err := survey.AskOne(&survey.Input{
			Message: "Username:",
		}, &credentials.Username, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	if len(credentials.Password) <= 0 {
		err := survey.AskOne(&survey.Password{
			Message: "Password:",
		}, &credentials.Password, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	if remember {
		if err := rememberCredentials(cfg, credentials); err != nil {
			return fmt.Errorf("Failed to store credentials: %+v", err)
		}
	}

	loginOpts := openapi.LoginRequest{
		ProviderName: cfg.Provider,
		Username:     openapi.PtrString(credentials.Username),
		Password:     openapi.PtrString(credentials.Password),
		DeviceId:     cfg.DeviceID,
	}
	ctx := context.Background()
	acceptHeaderFormatString := "application/vnd.appgate.peer-v%d+json"
	// initial authtentication, this will fail, since we will use the login response
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
	if len(providers) > 1 {
		prompt := &survey.Select{
			Message: "Choose a provider:",
			Options: providers,
		}
		if err := survey.AskOne(prompt, &loginOpts.ProviderName); err != nil {
			return err
		}
	}

	loginResponse, _, err := authenticator.Authentication(ctxWithAccept, loginOpts)
	if err != nil {
		return err
	}
	authToken := fmt.Sprintf("Bearer %s", loginResponse.GetToken())
	_, err = authenticator.Authorization(ctxWithAccept, authToken)
	if errors.Is(err, ErrPreConditionFailed) {
		otp, err := authenticator.InitializeOTP(ctxWithAccept, loginOpts.GetPassword(), authToken)
		if err != nil {
			return err
		}
		testOTP := func() (*openapi.LoginResponse, error) {
			var answer string
			optKey := &survey.Password{
				Message: "Please enter your one-time password:",
			}
			if err := survey.AskOne(optKey, &answer, survey.WithValidator(survey.Required)); err != nil {
				return nil, err
			}
			return authenticator.PushOTP(ctxWithAccept, answer, authToken)
		}
		// TODO add support for RadiusChallenge, Push
		switch otpType := otp.GetType(); otpType {
		case "Secret":
			barcodeFile, err := BarcodeHTMLfile(otp.GetBarcode(), otp.GetSecret())
			if err != nil {
				return err
			}
			fmt.Printf("\nOpen %s to scan the barcode to your authenticator app\n", barcodeFile.Name())
			fmt.Printf("\nIf you can’t use the barcode, enter %s in your authenticator app\n", otp.GetSecret())
			if err := browser.OpenURL(barcodeFile.Name()); err != nil {
				return err
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
						fmt.Println(err)
						continue
					}
				}
				if newToken != nil {
					authToken = fmt.Sprintf("Bearer %s", newToken.GetToken())
					break
				}
			}
		}
	} else if err != nil {
		return err
	}
	authorizationToken, err := authenticator.Authorization(ctxWithAccept, authToken)
	if err != nil {
		return err
	}

	cfg.BearerToken = authorizationToken.GetToken()
	cfg.ExpiresAt = authorizationToken.Expires.String()

	viper.Set("bearer", cfg.BearerToken)
	viper.Set("expires_at", cfg.ExpiresAt)
	viper.Set("url", cfg.URL)

	a, err := f.Appliance(cfg)
	if err != nil {
		return err
	}
	allAppliances, err := a.List(ctxWithAccept, nil)
	if err != nil {
		return err
	}
	primaryController, err := appliancepkg.FindPrimaryController(allAppliances, host)
	if err != nil {
		return err
	}
	stats, _, err := a.Stats(ctxWithAccept)
	if err != nil {
		return err
	}
	v, err := appliancepkg.GetPrimaryControllerVersion(*primaryController, stats)
	if err != nil {
		return err
	}
	viper.Set("primary_controller_version", v.String())
	if saveConfig {
		if err := viper.WriteConfig(); err != nil {
			return err
		}
	}
	return nil
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
		{
			Name: "path",
			Prompt: &survey.Input{
				Message: "Path to credentials file:",
				Default: fmt.Sprintf("%s/credentials", configuration.ConfigDir()),
			},
			Validate: survey.Required,
		},
	}

	answers := struct {
		Remember string `survey:"remember"`
		Path     string
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

	// Allow variable expansion for path
	cfg.CredentialsFile = os.ExpandEnv(answers.Path)

	if err := cfg.StoreCredentials(credentialsCopy); err != nil {
		return err
	}

	return nil
}
