package prompt

import "testing"

func TestPasswordConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		wantErr  bool
		askStubs func(*AskStubber)
	}{
		{
			name:    "same passwords",
			want:    "the_password",
			wantErr: false,
			askStubs: func(s *AskStubber) {
				s.StubOne("the_password") // password
				s.StubOne("the_password") // password confirmation
			},
		},
		{
			name:    "incorrect password",
			want:    "the_password",
			wantErr: true,
			askStubs: func(s *AskStubber) {
				s.StubOne("the_password")                  // password
				s.StubOne("inCorrectPasswordConfirmation") // password confirmation
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubber, teardown := InitAskStubber()
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
