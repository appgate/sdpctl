package prompt

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestPasswordConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		wantErr  bool
		askStubs func(*PromptStubber)
	}{
		{
			name:    "same passwords",
			want:    "the_password",
			wantErr: false,
			askStubs: func(s *PromptStubber) {
				s.StubOne("the_password") // password
				s.StubOne("the_password") // password confirmation
			},
		},
		{
			name:    "incorrect password",
			want:    "the_password",
			wantErr: true,
			askStubs: func(s *PromptStubber) {
				s.StubOne("the_password")                  // password
				s.StubOne("inCorrectPasswordConfirmation") // password confirmation
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubber, teardown := InitStubbers(t)
			defer teardown()
			tt.askStubs(stubber)
			got, err := PasswordConfirmation("")
			if (err != nil) != tt.wantErr {
				t.Errorf("PasswordConfirmation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PasswordConfirmation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPassPhrase(t *testing.T) {
	type args struct {
		stdIn     io.Reader
		canPrompt bool
		hasStdin  bool
	}
	tests := []struct {
		name     string
		args     args
		askStubs func(*PromptStubber)
		want     string
		wantErr  bool
	}{
		{
			name: "with stdin",
			args: args{
				stdIn:     bytes.NewBuffer([]byte("hunter2\n")),
				canPrompt: false,
				hasStdin:  true,
			},
			want:    "hunter2",
			wantErr: false,
		},
		{
			name: "with prompt",
			args: args{
				canPrompt: true,
				hasStdin:  false,
			},
			want:    "secret",
			wantErr: false,
			askStubs: func(s *PromptStubber) {
				s.StubPrompt("prompt message").AnswerWith("secret")
				s.StubPrompt("Confirm your passphrase:").AnswerWith("secret")
			},
		},
		{
			name: "no stdin no prompt",
			args: args{
				canPrompt: false,
				hasStdin:  false,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubber, teardown := InitStubbers(t)
			defer teardown()
			if tt.askStubs != nil {
				tt.askStubs(stubber)
			}
			got, err := GetPassphrase(tt.args.stdIn, tt.args.canPrompt, tt.args.hasStdin, "prompt message")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPassphrase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetPassphrase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateBackupPassphrase(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid alphanumeric",
			passphrase: "Password123",
			wantErr:    false,
		},
		{
			name:       "valid with special characters",
			passphrase: "P@ssw0rd!",
			wantErr:    false,
		},
		{
			name:       "valid complex passphrase",
			passphrase: "Myp@ssw0rd_123#",
			wantErr:    false,
		},
		{
			name:       "empty passphrase",
			passphrase: "",
			wantErr:    true,
			errMsg:     "passphrase cannot be empty",
		},
		{
			name:       "contains space",
			passphrase: "pass word",
			wantErr:    true,
			errMsg:     PassphraseInvalidMessage,
		},
		{
			name:       "contains tab",
			passphrase: "pass\tword",
			wantErr:    true,
			errMsg:     PassphraseInvalidMessage,
		},
		{
			name:       "contains emoji",
			passphrase: "password😀",
			wantErr:    true,
			errMsg:     PassphraseInvalidMessage,
		},
		{
			name:       "contains unicode",
			passphrase: "pässwörd",
			wantErr:    true,
			errMsg:     PassphraseInvalidMessage,
		},
		{
			name:       "all allowed special characters",
			passphrase: "!@#$%^&*()_+-=[]{}|;':\",./<>?~",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBackupPassphrase(tt.passphrase)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBackupPassphrase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateBackupPassphrase() error message = %v, want to contain %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestGetBackupPassphrase(t *testing.T) {
	tests := []struct {
		name      string
		stdIn     io.Reader
		canPrompt bool
		hasStdin  bool
		askStubs  func(*PromptStubber)
		want      string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid passphrase from stdin",
			stdIn:     bytes.NewBuffer([]byte("ValidPass123\n")),
			canPrompt: false,
			hasStdin:  true,
			want:      "ValidPass123",
			wantErr:   false,
		},
		{
			name:      "invalid passphrase from stdin with space",
			stdIn:     bytes.NewBuffer([]byte("Invalid Pass\n")),
			canPrompt: false,
			hasStdin:  true,
			want:      "",
			wantErr:   true,
			errMsg:    PassphraseInvalidMessage,
		},
		{
			name:      "invalid passphrase from stdin with emoji",
			stdIn:     bytes.NewBuffer([]byte("password😀\n")),
			canPrompt: false,
			hasStdin:  true,
			want:      "",
			wantErr:   true,
			errMsg:    PassphraseInvalidMessage,
		},
		{
			name:      "valid passphrase from prompt",
			canPrompt: true,
			hasStdin:  false,
			want:      "ValidPass123",
			wantErr:   false,
			askStubs: func(s *PromptStubber) {
				s.StubPrompt("prompt message").AnswerWith("ValidPass123")
				s.StubPrompt("Confirm your passphrase:").AnswerWith("ValidPass123")
			},
		},
		{
			name:      "no stdin no prompt",
			canPrompt: false,
			hasStdin:  false,
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubber, teardown := InitStubbers(t)
			defer teardown()
			if tt.askStubs != nil {
				tt.askStubs(stubber)
			}
			got, err := GetBackupPassphrase(tt.stdIn, tt.canPrompt, tt.hasStdin, "prompt message")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBackupPassphrase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("GetBackupPassphrase() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				return
			}
			if got != tt.want {
				t.Errorf("GetBackupPassphrase() = %v, want %v", got, tt.want)
			}
		})
	}
}
