package main

import (
	"os"

	"github.com/appgate/appgatectl/cmd"
)

func main() {
	code := cmd.Execute()
	os.Exit(int(code))
}
