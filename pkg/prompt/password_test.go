package prompt

import (
	"bytes"
	"io"
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
