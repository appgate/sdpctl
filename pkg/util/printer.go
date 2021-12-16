package util

import (
	"encoding/json"
	"fmt"
	"github.com/cheynewallace/tabby"
	"io"
	"text/tabwriter"
)

const (
	MinWidth = 0
	TabWidth = 0
	Padding  = 2
	PadChar  = ' '
)

func NewPrinter(output io.Writer) *tabby.Tabby {
	w := tabwriter.NewWriter(output, MinWidth, TabWidth, Padding, PadChar, tabwriter.DiscardEmptyColumns)
	return tabby.NewCustom(w)
}

func PrintJson(output io.Writer, v interface{}) error {
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
