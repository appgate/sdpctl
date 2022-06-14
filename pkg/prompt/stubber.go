package prompt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
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

type PromptStub struct {
	Value   interface{}
	Default bool
}
type AskStubber struct {
	stubs []*QuestionStub
}

func compareOptions(expected, got []string) error {
	if len(expected) != len(got) {
		return fmt.Errorf("expected %v, got %v (length mismatch)", expected, got)
	}
	for i, v := range expected {
		if v != got[i] {
			return fmt.Errorf("expected %v, got %v", expected, got)
		}
	}
	return nil
}
func InitAskStubber(t *testing.T) (*AskStubber, func()) {
	t.Helper()

	origSurveyAskOne := SurveyAskOne
	as := AskStubber{}

	answerFromStub := func(p survey.Prompt, fieldName string, response interface{}) error {
		var message string
		var defaultValue interface{}
		var options []string
		switch pt := p.(type) {
		case *survey.Confirm:
			message = pt.Message
			defaultValue = pt.Default
		case *survey.Input:
			message = pt.Message
			defaultValue = pt.Default
		case *survey.Select:
			message = pt.Message
			options = pt.Options
		case *survey.MultiSelect:
			message = pt.Message
			options = pt.Options
		case *survey.Password:
			message = pt.Message
		default:
			return fmt.Errorf("prompt type %T is not supported by the stubber", pt)
		}

		var stub *QuestionStub
		for _, s := range as.stubs {
			if !s.matched && (s.message == "" && strings.EqualFold(s.Name, fieldName) || s.message == message) {
				stub = s
				stub.matched = true
				break
			}
		}
		if stub == nil {
			return fmt.Errorf("no prompt stub for %q", message)
		}

		if len(stub.options) > 0 {
			if err := compareOptions(stub.options, options); err != nil {
				return fmt.Errorf("stubbed options mismatch for %q: %v", message, err)
			}
		}

		userValue := stub.Value

		if stringValue, ok := stub.Value.(string); ok && len(options) > 0 {
			foundIndex := -1
			for i, o := range options {
				if o == stringValue {
					foundIndex = i
					break
				}
			}
			if foundIndex < 0 {
				return fmt.Errorf("answer %q not found in options for %q: %v", stringValue, message, options)
			}
			userValue = core.OptionAnswer{
				Value: stringValue,
				Index: foundIndex,
			}
		}

		if stub.Default {
			if defaultIndex, ok := defaultValue.(int); ok && len(options) > 0 {
				userValue = core.OptionAnswer{
					Value: options[defaultIndex],
					Index: defaultIndex,
				}
			} else if defaultValue == nil && len(options) > 0 {
				userValue = core.OptionAnswer{
					Value: options[0],
					Index: 0,
				}
			} else {
				userValue = defaultValue
			}
		}

		if err := core.WriteAnswer(response, fieldName, userValue); err != nil {
			topic := fmt.Sprintf("field %q", fieldName)
			if fieldName == "" {
				topic = fmt.Sprintf("%q", message)
			}
			return fmt.Errorf("AskStubber failed writing the answer for %s: %w", topic, err)
		}
		return nil
	}

	SurveyAskOne = func(p survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
		return answerFromStub(p, "", response)
	}

	teardown := func() {
		SurveyAskOne = origSurveyAskOne
		for _, s := range as.stubs {
			if !s.matched {
				t.Errorf("unmatched prompt stub: %+v", s)
			}
		}
	}
	return &as, teardown
}

func (as *AskStubber) StubPrompt(msg string) *QuestionStub {
	stub := &QuestionStub{message: msg}
	as.stubs = append(as.stubs, stub)
	return stub
}
func (as *AskStubber) Stub(stubbedQuestions []*QuestionStub) {
	as.stubs = append(as.stubs, stubbedQuestions...)
}

func (as *AskStubber) StubOne(value interface{}) {
	as.Stub([]*QuestionStub{{Value: value}})
}
