package collective

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
)

type commandOpts struct {
	Out io.Writer
}

// NewCollectiveCmd return a new collective subcommand
func NewCollectiveCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use: "collective",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		TraverseChildren: true,
		Short:            "",
		Long:             "",
	}
	opts := &commandOpts{
		Out: f.IOOutWriter,
	}
	cmd.AddCommand(NewListCmd(opts))
	cmd.AddCommand(NewAddCmd(opts))
	cmd.AddCommand(NewDeleteCmd(opts))
	cmd.AddCommand(NewSetCmd(opts))

	return cmd
}

// readConfig read the config file from the a collective settings directory
// it tries to respect environment variable and parse boolean values correctly
//
// See: https://github.com/spf13/viper/issues/937
func readConfig(path string) (*configuration.Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw := make(map[string]interface{})
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, err
	}
	if v, ok := raw["insecure"].(string); ok {
		boolValue, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("insecure should be true|false, got %s %s", v, err)
		}
		delete(raw, "insecure")
		raw["insecure"] = boolValue
	}
	if v, ok := raw["debug"].(string); ok {
		boolValue, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("debug should be true|false, got %s %s", v, err)
		}
		delete(raw, "debug")
		raw["debug"] = boolValue
	}
	var config configuration.Config
	if err := mapstructure.Decode(raw, &config); err != nil {
		return nil, fmt.Errorf("%s file is corrupt: %s \n", path, err)
	}
	return &config, nil
}
