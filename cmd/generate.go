package cmd

import (
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Hidden:  true,
	Annotations: map[string]string{
		"skipAuthCheck": "true",
	},
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"man", "markdown", "md", "all"},
	Args:                  cobra.ExactValidArgs(1),
	Short:                 "Generates man pages for sdpctl",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "man":
			return generateManPages(cmd)
		case "markdown", "md":
			return generateMarkdown(cmd)
		case "all":
			var errs error
			if err := generateManPages(cmd); err != nil {
				errs = multierror.Append(err)
			}
			if err := generateMarkdown(cmd); err != nil {
				errs = multierror.Append(err)
			}
			return errs
		default:
			return errors.New("Invalid argument")
		}
	},
}

func generateManPages(cmd *cobra.Command) error {
	o := runtime.GOOS
	if o != "linux" && o != "darwin" {
		return errors.New("Man pages not available for this OS.")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.FromSlash(fmt.Sprintf("%s/build/man", cwd))
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}

	err = doc.GenManTree(cmd.Root(), &doc.GenManHeader{
		Title:   "SDPCTL",
		Section: "3",
	}, path)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, f := range files {
		out, err := os.Create(fmt.Sprintf("%s/%s.gz", path, f.Name()))
		if err != nil {
			return err
		}
		defer out.Close()

		out.Chmod(0644)
		w, err := gzip.NewWriterLevel(out, flate.BestCompression)
		if err != nil {
			return err
		}
		defer w.Close()
		b, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", path, f.Name()))
		if err != nil {
			return err
		}
		w.Write(b)
		os.Remove(fmt.Sprintf("%s/%s", path, f.Name()))
	}

	return nil
}

func generateMarkdown(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.FromSlash(fmt.Sprintf("%s/docs", cwd))
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}

	return doc.GenMarkdownTree(cmd.Root(), path)
}
