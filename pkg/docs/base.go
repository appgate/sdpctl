package docs

import (
	"fmt"
	"strings"
)

type ExampleDoc struct {
	Description string
	Command     string
	Output      string
}

type CommandDoc struct {
	Long     string
	Short    string
	Examples []ExampleDoc
}

func (e *ExampleDoc) String() string {
	exampleStrings := []string{}
	if len(e.Description) > 0 {
		exampleStrings = append(exampleStrings, fmt.Sprintf("  # %s", e.Description))
	}
	if len(e.Command) > 0 {
		exampleStrings = append(exampleStrings, fmt.Sprintf("  $ %s", e.Command))
	}
	if len(e.Output) > 0 {
		outString := strings.Split(e.Output, "\n")
		for _, o := range outString {
			exampleStrings = append(exampleStrings, fmt.Sprintf("  %s", o))
		}
	}
	return strings.Join(exampleStrings, "\n")
}

func (c *CommandDoc) ExampleString() string {
	examples := []string{}
	for _, e := range c.Examples {
		examples = append(examples, e.String())
	}
	return strings.Join(examples, "\n\n")
}
