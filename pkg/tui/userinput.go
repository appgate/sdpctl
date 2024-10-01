package tui

import "fmt"
import "os"
import "bufio"
import "strings"

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
	fmt.Print("(default: " + defaultvalue + ") ")
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
