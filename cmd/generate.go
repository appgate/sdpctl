package cmd

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
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
	ValidArgs:             []string{"man", "markdown", "md", "html", "all"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Short:                 "Generates man pages for sdpctl",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "man":
			return generateManPages(cmd)
		case "markdown", "md":
			return generateMarkdown(cmd)
		case "html":
			return generateHTML(cmd)
		case "all":
			var errs error
			if err := generateManPages(cmd); err != nil {
				errs = multierror.Append(err)
			}
			if err := generateMarkdown(cmd); err != nil {
				errs = multierror.Append(err)
			}
			if err := generateHTML(cmd); err != nil {
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
		return errors.New("Man pages not available for this OS")
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

	files, err := os.ReadDir(path)
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
		b, err := os.ReadFile(fmt.Sprintf("%s/%s", path, f.Name()))
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

func generateHTML(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.FromSlash(fmt.Sprintf("%s/docs", cwd))
	return filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		regex := regexp.MustCompile(`\.md$`)
		if !regex.MatchString(info.Name()) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		opts := html.RendererOptions{
			Flags:          html.FlagsNone,
			RenderNodeHook: renderHook,
		}
		renderer := html.NewRenderer(opts)
		output := string(markdown.ToHTML(b, nil, renderer))

		t := template.New("Render")
		t, err = t.Parse(htmlHeader + `{{.}}` + htmlFooter)
		t = template.Must(t, err)

		var processed bytes.Buffer
		if err := t.Execute(&processed, output); err != nil {
			return err
		}

		// Hack because markdown mishandles code blocks in the renderHook
		processedString := strings.ReplaceAll(processed.String(), `<pre>`, `<pre class="code-editor margin-bottom">`)
		processed = *bytes.NewBufferString(processedString)

		htmlName := strings.Replace(path, ".md", ".html", 1)
		if err := os.WriteFile(htmlName, processed.Bytes(), 0644); err != nil {
			return err
		}
		return nil
	})
}

func renderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if _, ok := node.(*ast.Heading); ok {
		level := strconv.Itoa(node.(*ast.Heading).Level)

		if entering {
			if level == "3" {
				w.Write([]byte(`<hr class="margin-bottom"><h3 class="emphasize text-left margin-bottom-small">`))
			} else {
				w.Write([]byte(fmt.Sprintf(`<h%s class="text-center margin-bottom">`, level)))
			}
		} else {
			w.Write([]byte(fmt.Sprintf(`</h%s>`, level)))
		}

		return ast.GoToNext, true
	} else if _, ok := node.(*ast.Link); ok {
		href := string(node.(*ast.Link).Destination)

		if entering {
			htmlRef := strings.Replace(href, ".md", ".html", 1)
			w.Write([]byte(fmt.Sprintf(`<a href="%s">`, htmlRef)))
		} else {
			w.Write([]byte("</a>"))
		}
		return ast.GoToNext, true
	} else if p, ok := node.(*ast.Paragraph); ok {
		if _, ok := p.GetParent().(*ast.Link); ok {
			return ast.GoToNext, false
		}
		if entering {
			if _, ok := p.GetChildren()[0].(*ast.Link); ok {
				w.Write([]byte(`<p>`))
			} else {
				w.Write([]byte(`<p class="margin-bottom">`))
			}
		} else {
			w.Write([]byte(`</p>`))
		}
		return ast.GoToNext, true
	}
	return ast.GoToNext, false
}

const (
	htmlHeader string = `<head>
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta name="Description" content="Appgate sdpctl Quick Start Guide">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="expires" content="0">
  <title>sdpctl Reference Guide</title>
  <link rel="stylesheet" href="./assets/guide.css">
</head>
<body>
  <main class="page text-center">
    <div class="box">
      <img src="./assets/appgate.svg" class="appgate-logo" alt="appgate inc logo">
      <h1 class="margin-top-small">sdpctl Reference Guide</h1>
      <hr>
      <div class="content text-left">
`

	htmlFooter string = `
      </div>
    </div>
  </main>
</body>
</html>`
)
