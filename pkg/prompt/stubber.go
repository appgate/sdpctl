package prompt

import (
	"errors"
	"fmt"
	"testing"
)

type QuestionStub struct {
	Name    string
	Value   interface{}
	Default bool

	matched bool
	message string
	options []string
}

func (s *QuestionStub) AnswerWith(v interface{}) *QuestionStub {
	s.Value = v
	return s
}
func (s *QuestionStub) AnswerDefault() *QuestionStub {
	s.Default = true
	return s
}

func compareOptions(expected, got []string) error {
	if len(expected) != len(got) {
		return fmt.Errorf("Expected %v, got %v (length mismatch)", expected, got)
	}
	for i, v := range expected {
		if v != got[i] {
			return fmt.Errorf("Expected %v, got %v", expected, got)
		}
	}
	return nil
}

var ErrNoPrompt = errors.New("No prompt stub")

type PromptStubber struct {
	stubs []*QuestionStub
}

func InitStubbers(t *testing.T) (*PromptStubber, func()) {
	origPromptPassword := PromptPassword
	origPromptInput := PromptInput
	origPromptConfirm := PromptConfirm
	origPromptSelection := PromptSelection
	origPromptMultiSelect := PromptMultiSelection
	origPromptSelectionIndex := PromptSelectionIndex
	ps := PromptStubber{}

	answerPromptStub := func(message string, choices []string) ([]string, int, error) {

		var stub *QuestionStub
		for _, s := range ps.stubs {
			if !s.matched && (s.message == "" || s.message == message) {
				stub = s
				stub.matched = true
				break
			}
		}
		if stub == nil {
			return nil, -1, fmt.Errorf("%w for %q", ErrNoPrompt, message)
		}

		if len(stub.options) > 0 {
			if err := compareOptions(stub.options, choices); err != nil {
				return nil, -1, fmt.Errorf("Stubbed options mismatch for %q: %v", message, err)
			}
		}

		userValue := stub.Value
		returnIndex := -1
		if stringValue, ok := stub.Value.(string); ok && len(choices) > 0 {
			foundIndex := -1
			for i, o := range choices {
				if o == stringValue {
					foundIndex = i
					returnIndex = i
					break
				}
			}
			if foundIndex < 0 {
				return nil, -1, fmt.Errorf("Answer %q not found in options for %q: %v", stringValue, message, choices)
			}
			userValue = stringValue
		}

		if stub.Default {
			userValue = choices[0]
			return []string{userValue.(string)}, 0, nil
		}
		var returnValue []string
		switch st := userValue.(type) {
		case string:
			returnValue = []string{st}
		case []string:
			returnValue = st
		case int:
			returnIndex = st
		case bool:
			if st {
				returnValue = []string{"yes"}
			} else {
				returnValue = []string{"no"}
			}
		}

		return returnValue, returnIndex, nil
	}
	PromptPassword = func(message string) (string, error) {
		answer, _, err := answerPromptStub(message, nil)
		returnValue := ""
		if len(answer) > 0 {
			returnValue = answer[0]
		}
		return returnValue, err
	}
	PromptInput = func(message string) (string, error) {
		answer, _, err := answerPromptStub(message, nil)
		returnValue := ""
		if len(answer) > 0 {
			returnValue = answer[0]
		}
		return returnValue, err
	}
	PromptConfirm = func(message string, defaultValue bool) (bool, error) {
		answer, _, err := answerPromptStub(message, nil)
		returnValue := defaultValue
		if len(answer) > 0 {
			returnValue = isAffirmative(answer[0])
		}
		return returnValue, err
	}
	PromptSelection = func(message string, choices []string, preSelected string) (string, error) {
		answer, _, err := answerPromptStub(message, choices)
		returnValue := ""
		if len(answer) > 0 {
			returnValue = answer[0]
		}
		return returnValue, err
	}
	PromptMultiSelection = func(message string, choices, preSelected []string) ([]string, error) {
		answer, _, err := answerPromptStub(message, choices)
		return answer, err
	}
	PromptSelectionIndex = func(message string, choices []string, preSelected string) (int, error) {
		_, answer, err := answerPromptStub(message, choices)
		return answer, err
	}
	teardown := func() {
		PromptPassword = origPromptPassword
		PromptInput = origPromptInput
		PromptConfirm = origPromptConfirm
		PromptSelection = origPromptSelection
		PromptMultiSelection = origPromptMultiSelect
		PromptSelectionIndex = origPromptSelectionIndex
		for _, s := range ps.stubs {
			if !s.matched {
				t.Errorf("Unmatched prompt stub: %+v", s)
			}
		}
	}
	return &ps, teardown
}

func (ps *PromptStubber) StubPrompt(msg string) *QuestionStub {
	stub := &QuestionStub{message: msg}
	ps.stubs = append(ps.stubs, stub)
	return stub
}

func (ps *PromptStubber) StubOne(value interface{}) {
	ps.Stub([]*QuestionStub{{Value: value}})
}

func (ps *PromptStubber) Stub(stubbedQuestions []*QuestionStub) {
	ps.stubs = append(ps.stubs, stubbedQuestions...)
}
