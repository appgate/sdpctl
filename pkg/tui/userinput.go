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

		value, _ := strconv.Atoi(text)
		if (err == nil) && (value >= 0) && (value < len(choices)) {
			return value, nil
		}
	}
}
