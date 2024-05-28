package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// Getenv returns environment variable value, if it does not exist, return fallback
func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func AppendIfMissing[V comparable](slice []V, i V) []V {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func InSlice(needle string, haystack []string) bool {
	stack := make([]string, len(haystack))
	copy(stack, haystack)
	sort.Strings(stack)
	i := sort.Search(
		len(stack),
		func(i int) bool { return stack[i] >= needle },
	)
	if i < len(stack) && stack[i] == needle {
		return true
	}
	return false
}

// SearchSlice will search a slice of strings and return all matching results.
// The search can either be case sensitive or not.
func SearchSlice(needle string, haystack []string, caseInsensitive bool) []string {
	result := []string{}

	searchTerm := needle
	if caseInsensitive {
		searchTerm = "(?i)" + searchTerm
	}
	regex := regexp.MustCompile(searchTerm)

	for _, s := range haystack {
		if regex.MatchString(s) {
			result = append(result, s)
		}
	}

	return result
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

func NormalizeURL(u string) (*url.URL, error) {
	if len(u) <= 0 {
		return nil, errors.New("no address set")
	}
	if r := regexp.MustCompile(`^https?://`); !r.MatchString(u) {
		u = fmt.Sprintf("https://%s", u)
	}
	url, err := url.ParseRequestURI(u)
	if err != nil {
		return nil, err
	}
	if url.Scheme != "https" {
		url.Scheme = "https"
	}
	return url, nil
}

func ParseFilteringFlags(flags *pflag.FlagSet, defaultFilter map[string]map[string]string) (map[string]map[string]string, []string, bool) {
	result := defaultFilter

	for v := range result {
		if arg, err := flags.GetStringToString(v); err == nil {
			if len(arg) > 0 {
				for f, value := range arg {
					result[v][f] = value
				}
			}
		} else {
			log.WithError(err).Error("failed to parse filter")
		}
	}

	orderBy, _ := flags.GetStringSlice("order-by")
	descending, _ := flags.GetBool("descending")

	return result, orderBy, descending
}

func IsUUID(str string) bool {
	_, err := uuid.Parse(str)
	return err == nil
}

func PrefixStringLines(s, prefixChar string, prefixLength int) string {
	split := strings.Split(s, "\n")
	for i, l := range split {
		if len(l) > 0 {
			split[i] = strings.Repeat(string(prefixChar), prefixLength) + l
		}
	}
	return strings.Join(split, "\n")
}

func Reverse[S ~[]T, T any](items S) S {
	if len(items) <= 1 {
		return items
	}
	result := make([]T, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		result = append(result, items[i])
	}
	return result
}

func ApplianceVersionString(v *version.Version) string {
	segments := v.Segments()
	res := fmt.Sprintf("%d.%d.%d-%s", segments[0], segments[1], segments[2], v.Metadata())
	preString := v.Prerelease()
	if len(preString) > 0 {
		res = res + "-" + preString
	}
	return res
}

func DockerTagVersion(v *version.Version) (string, error) {
	if envTag := os.Getenv("SDPCTL_DOCKER_TAG"); len(envTag) > 0 {
		return envTag, nil
	}
	if v == nil {
		return "", errors.New("DockerTagVersion() - version is nil")
	}
	segments := v.Segments()
	tagVersion := fmt.Sprintf("%d.%d", segments[0], segments[1])
	return tagVersion, nil
}

func AddSocketLogHook(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if stat.Mode().Type() != os.ModeSocket {
		return fmt.Errorf("%s is not a unix domain socket", path)
	}
	formatter := &log.JSONFormatter{
		FieldMap:        fieldMap,
		TimestampFormat: time.RFC3339,
	}
	hook, err := NewHook("unix", path, log.AllLevels, formatter)
	if err != nil {
		return err
	}
	log.AddHook(hook)
	return nil
}

func Find[T any](s []T, f func(T) bool) (T, error) {
	var r T
	for _, v := range s {
		if f(v) {
			return v, nil
		}
	}
	return r, errors.New("no match in slice")
}
