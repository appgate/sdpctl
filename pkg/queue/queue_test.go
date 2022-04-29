package queue

import (
	"errors"
	"testing"
)

var alphabet = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}

func TestQueueWithError(t *testing.T) {
	qw := New(len(alphabet), 2)
	for _, letter := range alphabet {
		qw.Push(letter)
	}
	err := qw.Work(func(v interface{}) error {
		if v.(string) == "e" {
			return errors.New("E is error letter")
		}
		return nil
	})
	if err == nil {
		t.Fatalf("Expected error, got none")
	}
}
func TestQueueNoError(t *testing.T) {
	qw := New(len(alphabet), 2)
	for _, letter := range alphabet {
		qw.Push(letter)
	}
	qw.queue.Unlock()
	err := qw.Work(func(v interface{}) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
}
