package auth

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSHACodeChallenge(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "string",
			args: args{
				s: "hello world",
			},
			want: "uU0nuZNNPgilLlLX2n2r-sSE7-N6U4DukIj3rOLvzek",
		},
		{
			name: "empty",
			args: args{
				s: "",
			},
			want: "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newSHACodeChallenge(tt.args.s); got != tt.want {
				t.Errorf("newSHACodeChallenge() = %q, want %v", got, tt.want)
			}
		})
	}
}

func TestOidcHandlerServeHTTPMissingCodeParameter(t *testing.T) {
	h := oidcHandler{
		TokenURL:     "http://auth-url.com",
		ClientID:     "abc123",
		CodeVerifier: "random-string",
		Response:     nil,
		errors:       make(chan error),
	}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	parentErr := make(chan error)
	go func(t *testing.T) {
		select {
		case err := <-h.errors:
			parentErr <- err
		case <-time.After(time.Second * 1):
			t.Errorf("expect error got none")
		}
	}(t)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.ServeHTTP)
	defer func() {
		defer close(parentErr)
		err := <-parentErr
		if err != ErrMissingCodePara {
			t.Fatalf("Expected %s got %s", ErrMissingCodePara, err)
		}

	}()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}
}
func TestRedirectHandler(t *testing.T) {
	h := redirectHandler{
		RedirectURL: "http://localhost",
	}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.ServeHTTP)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusSeeOther)
	}
}

func TestHTTPPostTokenURL(t *testing.T) {
	expected := `{
        "access_token":"abc123",
        "token_type":"Bearer",
        "refresh_token":"8xLOxBtZp8",
        "expires_in":3600,
        "id_token":"long-id-token"
    }`
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, expected)
	}))
	defer svr.Close()

	h := oidcHandler{
		TokenURL:     svr.URL,
		ClientID:     "abc123",
		CodeVerifier: "random-string",
		Response:     make(chan oIDCResponse),
		errors:       make(chan error),
	}
	response, err := h.httpPostTokenURL("foobar")
	if err != nil {
		t.Fatal(err)
	}
	if response.AccessToken != "abc123" {
		t.Errorf("wrong accessToken %s", response.AccessToken)
	}
	if response.TokenType != "Bearer" {
		t.Errorf("wrong token_type %s", response.TokenType)
	}
	if response.IDToken != "long-id-token" {
		t.Errorf("wrong id_token %s", response.IDToken)
	}
}

func TestHTTPPostTokenURLErrorResponse(t *testing.T) {
	expected := `{
        "error" : "invalid_client",
        "error_description" : "No client credentials found."
    }`
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, expected)
	}))
	defer svr.Close()

	h := oidcHandler{
		TokenURL:     svr.URL,
		ClientID:     "abc123",
		CodeVerifier: "random-string",
		Response:     make(chan oIDCResponse),
		errors:       make(chan error),
	}
	_, err := h.httpPostTokenURL("foobar")
	if err == nil {
		t.Fatal("expected err got none")
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Expected invalid request err got %s", err)
	}
}
