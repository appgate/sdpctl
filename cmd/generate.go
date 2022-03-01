/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:                   "generate",
	Aliases:               []string{"gen"},
	Hidden:                true,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"man"},
	Args:                  cobra.ExactValidArgs(1),
	Short:                 "Generates man pages for sdpctl",
	RunE: func(cmd *cobra.Command, args []string) error {
		o := runtime.GOOS
		if o != "linux" && o != "darwin" {
			return errors.New("Man pages not available for this OS.")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		manPath := filepath.FromSlash(fmt.Sprintf("%s/build/man", cwd))
		if err := os.MkdirAll(manPath, 0700); err != nil {
			return err
		}

		err = doc.GenManTree(cmd.Root(), &doc.GenManHeader{
			Title:   "SDPCTL",
			Section: "3",
		}, manPath)
		if err != nil {
			return err
		}

		files, err := ioutil.ReadDir(manPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			out, err := os.Create(fmt.Sprintf("%s/%s.gz", manPath, f.Name()))
			if err != nil {
				return err
			}
			defer out.Close()

			w := gzip.NewWriter(out)
			defer w.Close()
			b, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", manPath, f.Name()))
			if err != nil {
				return err
			}
			w.Write(b)
			os.Remove(fmt.Sprintf("%s/%s", manPath, f.Name()))
		}

		return nil
	},
}
