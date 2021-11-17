package prompt

import (
	"fmt"
	"reflect"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
)

type QuestionStub struct {
	Name    string
	Value   interface{}
	Default bool
}
type PromptStub struct {
	Value   interface{}
	Default bool
}
type AskStubber struct {
	AskOnes  []*survey.Prompt
	OneCount int
	StubOnes []*PromptStub
}

func InitAskStubber() (*AskStubber, func()) {
	origSurveyAskOne := SurveyAskOne
	as := AskStubber{}

	SurveyAskOne = func(p survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
		as.AskOnes = append(as.AskOnes, &p)
		count := as.OneCount
		as.OneCount += 1
		if count >= len(as.StubOnes) {
			panic(fmt.Sprintf("1-more asks than stubs. most recent call: %v", p))
		}
		stubbedPrompt := as.StubOnes[count]
		if stubbedPrompt.Default {
			// TODO this is failing for basic AskOne invocations with a string result.
			defaultValue := reflect.ValueOf(p).Elem().FieldByName("Default")
			_ = core.WriteAnswer(response, "", defaultValue)
		} else {
			_ = core.WriteAnswer(response, "", stubbedPrompt.Value)
		}

		return nil
	}
	teardown := func() {
		SurveyAskOne = origSurveyAskOne
	}
	return &as, teardown
}

func (as *AskStubber) StubOne(value interface{}) {
	as.StubOnes = append(as.StubOnes, &PromptStub{
		Value: value,
	})
}
