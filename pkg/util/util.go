package util

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"sort"

	"github.com/spf13/pflag"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

// Getenv returns environment variable value, if it does not exist, return fallback
func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func AppendIfMissing(slice []int, i int) []int {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func InSlice(needle string, haystack []string) bool {
	sort.Strings(haystack)
	i := sort.Search(len(haystack),
		func(i int) bool { return haystack[i] >= needle })
	if i < len(haystack) && haystack[i] == needle {
		return true
	}
	return false
}

func FileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func InBetween(i, min, max int) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
	}
}

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// IsValidURL tests a string to determine if it is a well-structured url or not.
func IsValidURL(addr string) bool {
	_, err := url.ParseRequestURI(addr)
	if err != nil {
		return false
	}

	u, err := url.Parse(addr)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return true
}

func ParseFilteringFlags(flags *pflag.FlagSet) map[string]map[string]string {
	result := map[string]map[string]string{
		"filter":  {},
		"exclude": {},
	}

	for v := range result {
		arg, err := flags.GetStringToString(v)
		if err == nil {
			result[v] = arg
		}
	}

	return result
}

func AddDefaultSpinner(p *mpb.Progress, name string, stage string, cmsg string, opts ...mpb.BarOption) *mpb.Bar {
	options := []mpb.BarOption{
		mpb.BarFillerOnComplete("✓"),
		mpb.BarWidth(1),
		mpb.AppendDecorators(
			decor.Name(name, decor.WCSyncWidthR),
			decor.Name(":", decor.WC{W: 2, C: decor.DidentRight}),
			decor.OnComplete(decor.OnAbort(decor.Name(stage), "failed"), cmsg),
		),
	}
	options = append(options, opts...)
	return p.AddSpinner(1, options...)
}
