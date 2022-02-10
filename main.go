package main

import (
	"os"

	"github.com/appgate/sdpctl/cmd"
)

func main() {
	code := cmd.Execute()
	os.Exit(int(code))
}
