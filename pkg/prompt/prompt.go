package prompt

import (
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// AskConfirmation make sure user confirm action, otherwise abort.
func AskConfirmation(m ...string) error {
	m = append(m, "Do you want to continue?")
	ok, err := PromptConfirm(strings.Join(m, "\n\n"), false)
	if err != nil || !ok {
		return cmdutil.ErrExecutionCanceledByUser
	}
	return nil
}

type (
	errMsg error
)

type textInputModel struct {
	textinput textinput.Model
	quitting  bool
	err       error
}

func newTextInputModel(message string) textInputModel {
	ta := textinput.New()
	ta.Prompt = message
	ta.Focus()
	ta.CharLimit = 1024

	return textInputModel{
		textinput: ta,
		err:       nil,
	}
}

func (m textInputModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textinput, tiCmd = m.textinput.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textinput.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			m.quitting = true
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m textInputModel) View() string {
	if m.quitting {
		m.textinput.Cursor.SetMode(cursor.CursorHide)
		return m.textinput.View() + "\n"
	}
	return m.textinput.View()
}

func isAffirmative(input string) bool {
	input = strings.ToLower(input)
	return input == "y" || input == "yes"
}

var PromptConfirm = func(message string, defaultValue bool) (bool, error) {
	defaultString := "y/N"
	if defaultValue {
		defaultString = "Y/n"
	}
	messageWithDefault := fmt.Sprintf("%s (%s): ", message, defaultString)
	m := newTextInputModel(messageWithDefault)
	p := tea.NewProgram(m)

	returnedModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
		return false, err
	}
	if returnedModel.(textInputModel).textinput.Value() == "" {
		return defaultValue, nil
	}
	return isAffirmative(returnedModel.(textInputModel).textinput.Value()), nil
}

var PromptPassword = func(message string) (string, error) {
	m := newTextInputModel(fmt.Sprintf("%s ", message))
	m.textinput.EchoMode = textinput.EchoPassword
	m.textinput.EchoCharacter, _ = utf8.DecodeLastRuneInString("*")
	p := tea.NewProgram(m)

	returnedModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return returnedModel.(textInputModel).textinput.Value(), nil
}

var PromptInputDefault = func(message, defaultValue string) (string, error) {
	m := newTextInputModel(fmt.Sprintf("%s ", message))
	m.textinput.SetValue(defaultValue)
	p := tea.NewProgram(m)

	returnedModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return returnedModel.(textInputModel).textinput.Value(), nil
}

var PromptInput = func(message string) (string, error) {
	return PromptInputDefault(message, "")
}
