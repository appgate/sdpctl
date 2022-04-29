package queue

import (
	"errors"
	"testing"
)

func TestQueue(t *testing.T) {
	alphabet := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	type args struct {
		capacity int
		workers  int
		closure  Closure
		items    []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "process alphabet queue with one error",
			args: args{
				capacity: len(alphabet),
				workers:  2,
				items:    alphabet,
				closure: func(v interface{}) error {
					if v.(string) == "e" {
						return errors.New("E is error letter")
					}
					return nil
				},
			},
			wantErr: true,
		},
		{
			name: "process alphabet queue with no errors",
			args: args{
				capacity: len(alphabet),
				workers:  2,
				items:    alphabet,
				closure: func(v interface{}) error {
					return nil
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qw := New(tt.args.capacity, tt.args.workers)
			for _, letter := range tt.args.items {
				qw.Push(letter)
			}
			err := qw.Work(tt.args.closure)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Expected %v got %v", tt.wantErr, err)
			}
		})
	}
}
