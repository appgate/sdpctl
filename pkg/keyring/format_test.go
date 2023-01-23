package keyring

import (
	"errors"
	"testing"
	"time"
)

func init() {
	// overwrite timeout in test only
	keyringTimeout = time.Millisecond * 500
}
func TestRunWithTimeout(t *testing.T) {
	type args struct {
		task func() error
	}
	tests := []struct {
		name                 string
		args                 args
		wantErr              bool
		wantTimeOutErr       bool
		environmentVariables map[string]string
	}{
		{
			name: "timeout reached",
			args: args{
				task: func() error {
					time.Sleep(keyringTimeout + 2*time.Second)
					return nil
				},
			},
			wantErr:        true,
			wantTimeOutErr: true,
		},
		{
			name: "no error",
			args: args{
				task: func() error {
					return nil
				},
			},
			wantErr:        false,
			wantTimeOutErr: false,
		},
		{
			name: "normal error",
			args: args{
				task: func() error {
					return errors.New("a error")
				},
			},
			wantErr:        true,
			wantTimeOutErr: false,
		},
		{
			name: "environment variable SDPCTL_NO_KEYRING set",
			args: args{
				task: func() error {
					time.Sleep(keyringTimeout + 10*time.Second)
					return nil
				},
			},
			environmentVariables: map[string]string{
				"SDPCTL_NO_KEYRING": "true",
			},
			wantErr:        false,
			wantTimeOutErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.environmentVariables {
				t.Setenv(k, v)
			}
			err := runWithTimeout(tt.args.task)
			if (err != nil) != tt.wantErr && errors.Is(err, ErrKeyringTimeOut) && !tt.wantTimeOutErr {
				t.Fatalf("got timeout error, did not expect it %s", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("runWithTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
