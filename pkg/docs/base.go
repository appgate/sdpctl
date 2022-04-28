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

// String formats each example to a single string and returns
func (e *ExampleDoc) String() string {
	exampleStrings := []string{}
	if len(e.Description) > 0 {
		eStrings := strings.Split(e.Description, "\n")
		for _, e := range eStrings {
			exampleStrings = append(exampleStrings, fmt.Sprintf("  # %s", e))
		}
	}
	if len(e.Command) > 0 {
		cmdLines := strings.Split(e.Command, "\n")
		for _, l := range cmdLines {
			exampleStrings = append(exampleStrings, fmt.Sprintf("  $ %s", l))
		}
	}
	if len(e.Output) > 0 {
		outString := strings.Split(e.Output, "\n")
		for _, o := range outString {
			exampleStrings = append(exampleStrings, fmt.Sprintf("  %s", o))
		}
	}
	return strings.Join(exampleStrings, "\n")
}

// ExampleString generates a single string from multiple entries of ExampleDoc for printing in the command specifications
func (c *CommandDoc) ExampleString() string {
	examples := []string{}
	for _, e := range c.Examples {
		examples = append(examples, e.String())
	}
	return strings.Join(examples, "\n\n")
}
