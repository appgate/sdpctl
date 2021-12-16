package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/sirupsen/logrus"
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

func ParseFilteringFlags(flags *pflag.FlagSet) map[string]map[string]string {
	result := map[string]map[string]string{
		"filter":  {},
		"exclude": {},
	}

    for v := range result {
        arg, err := flags.GetStringToString(v)
        if err != nil {
            logrus.Warnf("Failed to parse %s flag: %s", v, err)
        }
        result[v] = arg
    }

    return result
}

func FilterAppliances(appliances []openapi.Appliance, filter map[string]map[string]string) []openapi.Appliance {
	// apply normal filter
	if len(filter["filter"]) > 0 {
		appliances = applyFilter(appliances, filter["filter"])
	}

	if len(filter["exclude"]) <= 0 {
		return appliances
	}

	// apply exclusion filter
	filtered := []openapi.Appliance{}
	toExclude := applyFilter(appliances, filter["exclude"])
	for _, appliance := range appliances {
		aID := appliance.GetId()
		for _, exa := range toExclude {
			eID := exa.GetId()
			if aID != eID {
				filtered = append(filtered, appliance)
			}
		}
	}

	return filtered
}

func applyFilter(appliances []openapi.Appliance, filter map[string]string) []openapi.Appliance {
	var filteredAppliances []openapi.Appliance
	for _, a := range appliances {
		for k, s := range filter {
			switch k {
			case "name":
				regex := regexp.MustCompile(s)
				if regex.MatchString(a.GetName()) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "id":
				regex := regexp.MustCompile(s)
				if regex.MatchString(a.GetId()) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "tags", "tag":
				tagSlice := strings.Split(s, ",")
				appTags := a.GetTags()
				for _, t := range tagSlice {
					regex := regexp.MustCompile(t)
					for _, at := range appTags {
						if regex.MatchString(at) {
							filteredAppliances = append(filteredAppliances, a)
						}
					}
				}
			case "version":
				regex := regexp.MustCompile(s)
				version := a.GetVersion()
				versionString := fmt.Sprintf("%d", version)
				if regex.MatchString(versionString) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "hostname", "host":
				regex := regexp.MustCompile(s)
				if regex.MatchString(a.GetHostname()) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "active", "activated":
				b, err := strconv.ParseBool(s)
				if err != nil {
					logrus.Warnf("Failed to parse boolean filter value: %x", err)
				}
				if a.GetActivated() == b {
					filteredAppliances = append(filteredAppliances, a)
				}
			}
		}
	}
	return filteredAppliances
}
