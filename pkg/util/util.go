package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
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

var ErrMalformedURL error = errors.New("malformed url")
var ErrNotAURL error = errors.New("not a url")

// IsValidURL tests a string to determine if it is a well-structured url or not.
func IsValidURL(addr string) error {
	_, err := url.ParseRequestURI(addr)
	if err != nil {
		return err
	}
	if r := regexp.MustCompile(`https?://`); len(r.FindAllString(addr, -1)) > 1 {
		return fmt.Errorf("%w: '%s'", ErrMalformedURL, addr)
	}
	u, err := url.Parse(addr)
	if err != nil {
		return err
	}
	if u.Scheme == "" || u.Host == "" {
		return ErrNotAURL
	}
	return nil
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

func StringAbbreviate(s string) string {
	i := strings.Index(s, "\n")
	if i < 0 {
		return s
	}
	return s[:i] + " [...]"
}

func BaseAuthContext(token string) context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, openapi.ContextAccessToken, token)
}

func TokenFromConfig(token string, bearerToken *string) (string, error) {
	if token != "" {
		return token, nil
	}
	if bearerToken != nil {
		return *bearerToken, nil
	}
	return "", fmt.Errorf("Credentials are not set, please use the 'login' command to authenticate first")
}
