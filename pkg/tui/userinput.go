package tui

import "fmt"
import "os"
import "bufio"
import "strconv"
import "strings"
import "golang.org/x/crypto/ssh/terminal"

func YesNo(prompt string, defaultyes bool) bool {
	for {
		fmt.Print(prompt)

		if defaultyes {
			fmt.Print(" [Yn] ")
		} else {
			fmt.Print(" [yN] ")
		}

		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')

		if err != nil {
			return defaultyes
		}

		lowtext := strings.TrimRight(strings.ToLower(text), "\r\n ")

		if lowtext == "" {
			return defaultyes
		} else if lowtext == "yes" || lowtext == "y" {
			return true
		} else if lowtext == "no" || lowtext == "n" {
			return false
		}
	}
}

func Input(prompt string, defaultvalue string) (string, error) {
	fmt.Print(prompt)
	if len(defaultvalue) > 0 {
		fmt.Print("(default: " + defaultvalue + ") ")
	}
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	text = strings.TrimRight(text, "\r\n ")

	if text == "" {
		return defaultvalue, nil
	}

	return text, nil
}

func Password(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func Choice(prompt string, choices []string) (int, error) {
	for {
		for i, v := range choices {
			fmt.Printf("%d: %s\n", i, v)
		}

		fmt.Print(prompt)

		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}

		text = strings.Trim(text, "\r\n ")

		value, err := strconv.Atoi(text)
		if (err == nil) && (value >= 0) && (value < len(choices)) {
			return value, nil
		}
	}
}

func MultipleChoice(prompt string, choices []string) ([]int, error) {
	for {
		for i, v := range choices {
			fmt.Printf("%d: %s\n", i, v)
		}
		fmt.Print(prompt)

		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		text = strings.Trim(text, "\r\n ")

		if text == "" {
			return make([]int, 0), nil
		}

		selected := strings.Split(text, ",")

		canreturn := true

		var r []int
		r = make([]int, len(selected))

		for i, v := range selected {
			v = strings.Trim(v, "\r\n ")
			intval, err := strconv.Atoi(v)
			if err != nil {
				fmt.Println("Unable to parse as integer: " + v)
				canreturn = false
				break
			}
			if intval < 0 || intval >= len(choices) {
				fmt.Println("Value out of range selected")
				canreturn = false
				break
			}
			r[i] = intval
		}

		if canreturn {
			return r, nil
		}
	}
}
