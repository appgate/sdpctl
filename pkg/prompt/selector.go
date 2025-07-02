package prompt

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	appStyle          = lipgloss.NewStyle().Padding(0, 0)
	titleStyle        = lipgloss.NewStyle().MarginLeft(0)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(1)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("87"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(0, 0, 0, 0)
)

type item struct {
	name     string
	selected bool
}

func (i item) FilterValue() string { return i.name }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	var str string
	if i.selected {
		str = selectedItemStyle.Render(fmt.Sprintf("[x] %s", i.name))
	} else {
		str = fmt.Sprintf("[ ] %s", i.name)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(">" + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list          list.Model
	quitting      bool
	oneChoice     bool
	selectedIndex int
}

func (m model) Init() tea.Cmd {
	return nil
}

func itemIndex(items []list.Item, toMatch string) int {
	return slices.IndexFunc[[]list.Item, list.Item](items, func(comp list.Item) bool { return comp.(item).name == toMatch })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return m, tea.Interrupt
		case "ctrl+z":
			return m, tea.Suspend
		case "q":
			m.quitting = true
			return m, tea.Quit

		case "enter", " ":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				newState := !i.selected
				if keypress == "enter" {
					// Always select the item under the cursor when enter is pressed
					newState = true
				}
				newItem := item{
					name:     i.name,
					selected: newState,
				}
				absoluteIndex := itemIndex(m.list.Items(), i.name)
				cmd := m.list.SetItem(absoluteIndex, newItem)
				if m.oneChoice {
					if m.selectedIndex >= 0 && m.selectedIndex != absoluteIndex {
						lastSelected := m.list.Items()[m.selectedIndex].(item)
						newLastSelected := item{
							name:     lastSelected.name,
							selected: false,
						}
						m.list.SetItem(m.selectedIndex, newLastSelected)
					} else if m.selectedIndex == absoluteIndex {
						m.selectedIndex = -1
					}
					m.selectedIndex = absoluteIndex
				}
				if keypress == "enter" {
					cmd = tea.Quit
					m.quitting = true

				}
				return m, cmd
			}
			return m, nil
		case "left":
			modifyVisible(m, false)
		case "right":
			if !m.oneChoice {
				modifyVisible(m, true)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func modifyVisible(m model, selected bool) {
	for _, toModify := range m.list.VisibleItems() {
		if toModify.(item).selected != selected {
			newItem := item{
				name:     toModify.(item).name,
				selected: selected,
			}
			itemIndex := itemIndex(m.list.Items(), toModify.(item).name)
			m.list.SetItem(itemIndex, newItem)
		}
	}
}

func selectedItems(m model) []string {
	var selectedItems []string
	for _, i := range m.list.Items() {
		if i.(item).selected {
			selectedItems = append(selectedItems, i.(item).name)
		}
	}
	return selectedItems
}

func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render(m.list.Title + "\n" + strings.Join(selectedItems(m), "\n") + "\n")
	}
	return "\n" + m.list.View()
}

func makeItem(name string) item {
	return item{
		name:     name,
		selected: false,
	}
}

func prompt(question string, choices, preSelected []string, oneChoice bool) ([]string, error) {
	items := []list.Item{}
	selectedIndex := -1
	for _, choice := range choices {
		newItem := makeItem(choice)
		if preSelected != nil {
			if isPreselected := slices.Index(preSelected, newItem.name); isPreselected >= 0 {
				if oneChoice {
					selectedIndex = len(items)
				}
				newItem.selected = true
			}
		}
		items = append(items, newItem)
	}

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	enterAction := "continue"
	if oneChoice {
		enterAction = "select"
	}
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(
			key.WithKeys("enter"), key.WithHelp("enter", enterAction)),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select"))}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", enterAction)),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select"))}
	}

	l.Title = question
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.Help.Styles.ShortKey = helpStyle
	l.Help.Styles.ShortDesc = helpStyle
	l.Help.Styles.FullKey = helpStyle
	l.Help.Styles.FullDesc = helpStyle
	l.FilterInput.PromptStyle = selectedItemStyle
	l.FilterInput.Cursor.Style = itemStyle

	m := model{list: l, oneChoice: oneChoice, selectedIndex: selectedIndex}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		return nil, err
	}
	return selectedItems(m), nil
}

var PromptMultiSelection = func(question string, choices, preSelected []string) ([]string, error) {
	result, err := prompt(question, choices, preSelected, false)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}
	return nil, fmt.Errorf("No input provided")

}

var PromptSelection = func(question string, choices []string, preSelected string) (string, error) {
	var result []string
	var err error
	if preSelected == "" {
		result, err = prompt(question, choices, []string{}, true)
	} else {
		result, err = prompt(question, choices, []string{preSelected}, true)
	}
	if err != nil {
		return "", err
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return "", fmt.Errorf("No input provided")
}

var PromptMultiSelectIndex = func(question string, choices []string, preSelected []string) ([]int, error) {
	result, err := PromptMultiSelection(question, choices, preSelected)
	if err != nil {
		return nil, err
	}
	var selectedIndices []int
	j := 0
	for i := 0; i < len(choices) && j < len(result); i++ {
		if choices[i] == result[j] {
			selectedIndices = append(selectedIndices, i)
			j++
		}
	}
	return selectedIndices, nil
}

var PromptSelectionIndex = func(question string, choices []string, preSelected string) (int, error) {
	result, err := prompt(question, choices, []string{preSelected}, true)
	if err != nil {
		return -1, err
	}
	if len(result) > 0 {
		for i, name := range choices {
			if result[0] == name {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf(("No input provided"))
}
