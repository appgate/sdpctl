package util

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/cheynewallace/tabby"
)

const (
	MinWidth = 0
	TabWidth = 0
	PadChar  = ' '
)

func NewPrinter(output io.Writer, padding int) *tabby.Tabby {
	w := tabwriter.NewWriter(output, MinWidth, TabWidth, padding, PadChar, tabwriter.DiscardEmptyColumns)
	return tabby.NewCustom(w)
}

func PrintJSON(output io.Writer, v interface{}) error {
	j, err := json.MarshalIndent(&v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(j))
	if err != nil {
		return err
	}
	return nil
}
