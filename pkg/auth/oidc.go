package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/keyring"
	"github.com/google/uuid"
	"github.com/pkg/browser"
)

// oIDCResponse Successful Token Response body.
// https://openid.net/specs/openid-connect-core-1_0.html#TokenResponse
type oIDCResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// oIDCError is the Token error response body if we get HTTP 400-500 status code.
// https://openid.net/specs/openid-connect-core-1_0.html#AuthError
type oIDCError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorCodes       []int  `json:"error_codes"`
	Timestamp        string `json:"timestamp"`
	TraceID          string `json:"trace_id"`
	CorrelationID    string `json:"correlation_id"`
}

type OpenIDConnect struct {
	Factory    *factory.Factory
	Client     *openapi.APIClient
	httpServer *http.Server
	response   chan oIDCResponse
	errors     chan error
}

func NewOpenIDConnect(f *factory.Factory, client *openapi.APIClient) *OpenIDConnect {
	o := &OpenIDConnect{
		Factory: f,
		Client:  client,
	}
	o.response = make(chan oIDCResponse)
	o.errors = make(chan error)

	return o
}

func (o *OpenIDConnect) Close() {
	if o.httpServer != nil {
		o.httpServer.Close()
	}
}

type redirectHandler struct {
	RedirectURL string
}

func (h redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.RedirectURL, http.StatusSeeOther)
}

type oidcHandler struct {
	TokenURL, ClientID, CodeVerifier string
	Response                         chan oIDCResponse
	errors                           chan error
}

// oidcRedirectAddress is the local webserver for the redirect loop used with oidc provider
// it uses the same port as the client for consistency.
const (
	oidcPort            string = ":29001"
	oidcRedirectAddress string = "http://localhost" + oidcPort
)

var (
	ErrMissingCodePara = errors.New("missing code in parameter")
	ErrInvalidRequest  = errors.New("error response")
)

func (h oidcHandler) httpPostTokenURL(code string) (*oIDCResponse, error) {
	form := url.Values{}
	form.Add("client_id", h.ClientID)
	form.Add("grant_type", "authorization_code")
	form.Add("redirect_uri", oidcRedirectAddress+"/oidc")
	form.Add("code_verifier", h.CodeVerifier)
	form.Add("code", code)
	req, err := http.NewRequest(http.MethodPost, h.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var errResponse oIDCError
		if err = json.Unmarshal(body, &errResponse); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w %s %s", ErrInvalidRequest, errResponse.Error, errResponse.ErrorDescription)
	}

	var data oIDCResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (h oidcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if len(code) < 1 {
		w.WriteHeader(http.StatusInternalServerError)
		h.errors <- ErrMissingCodePara
		return
	}

	data, err := h.httpPostTokenURL(code)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if errors.Is(ErrInvalidRequest, err) {
			fmt.Fprint(w, err)
		}
		return
	}
	h.Response <- *data
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, OpenIDConnectHTML)
}

func newSHACodeChallenge(s string) string {
	hash := sha256.New()
	hash.Write([]byte(s))
	size := hash.Size()

	sum := hash.Sum(nil)[:size]
	return base64.RawURLEncoding.EncodeToString(sum)
}

func (o OpenIDConnect) refreshToken(clientID, tokenURL, refreshToken string) (*oIDCResponse, error) {
	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("grant_type", "refresh_token")
	form.Add("refresh_token", refreshToken)
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data oIDCResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

var ErrPlatformNotSupported = errors.New("Provider with OpenID Connect is not supported on your system")

func (o OpenIDConnect) signin(ctx context.Context, loginOpts openapi.LoginRequest, provider openapi.IdentityProvidersNamesGet200ResponseDataInner) (*signInResponse, error) {
	authenticator := NewAuth(o.Client)
	prefix, err := o.Factory.Config.KeyringPrefix()
	if err != nil {
		return nil, err
	}
	if k, err := keyring.GetRefreshToken(prefix); err == nil {
		t, err := o.refreshToken(provider.GetClientId(), provider.GetTokenUrl(), k)
		if err != nil {
			return nil, err
		}
		if t != nil && len(t.IDToken) > 0 && len(t.AccessToken) > 0 {
			loginOpts.IdToken = &t.IDToken
			loginOpts.AccessToken = &t.AccessToken

			loginResponse, _, err := authenticator.Authentication(ctx, loginOpts)
			if err != nil {
				return nil, err
			}

			response := &signInResponse{
				Token:     loginResponse.GetToken(),
				Expires:   time.Now().Local().Add(time.Second * time.Duration(t.ExpiresIn)),
				LoginOpts: &loginOpts,
			}
			return response, nil
		}
	}

	mux := http.NewServeMux()
	o.httpServer = &http.Server{
		Addr:    oidcPort,
		Handler: mux,
	}

	u, err := url.Parse(provider.GetAuthUrl())
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("scope", provider.GetScope())
	q.Set("client_id", provider.GetClientId())
	q.Set("state", "client")
	q.Set("redirect_uri", oidcRedirectAddress+"/oidc")
	codeVerifier := uuid.New().String()
	q.Set("code_challenge", newSHACodeChallenge(codeVerifier))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()

	mux.Handle("/", redirectHandler{
		RedirectURL: u.String(),
	})
	mux.Handle("/oidc", oidcHandler{
		Response:     o.response,
		errors:       o.errors,
		TokenURL:     provider.GetTokenUrl(),
		ClientID:     provider.GetClientId(),
		CodeVerifier: codeVerifier,
	})

	go func() {
		if err := o.httpServer.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintf(o.Factory.StdErr, "[error] %s\n", err)
			}
		}
	}()
	browser.Stderr = io.Discard
	if err := browser.OpenURL(oidcRedirectAddress); err != nil {
		return nil, ErrPlatformNotSupported
	}
	select {
	case err := <-o.errors:
		return nil, err
	case t := <-o.response:

		loginOpts.IdToken = &t.IDToken
		loginOpts.AccessToken = &t.AccessToken

		if err := keyring.SetRefreshToken(prefix, t.RefreshToken); err != nil {
			return nil, ErrPlatformNotSupported
		}

		loginResponse, _, err := authenticator.Authentication(ctx, loginOpts)
		if err != nil {
			return nil, err
		}

		response := &signInResponse{
			Token:     loginResponse.GetToken(),
			Expires:   time.Now().Local().Add(time.Second * time.Duration(t.ExpiresIn)),
			LoginOpts: &loginOpts,
		}
		return response, nil
	}
}
