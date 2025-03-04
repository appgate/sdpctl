package device

import (
	"io"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/device"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

type DeviceOptions struct {
	Config     *configuration.Config
	Out        io.Writer
	Device     func(c *configuration.Config) (*device.Device, error)
	Debug      bool
	useJSON    bool
	orderBy    []string
	descending bool
}

func NewDeviceCmd(f *factory.Factory) *cobra.Command {
	opts := &DeviceOptions{
		Config: f.Config,
		Out:    f.IOOutWriter,
		Device: f.Device,
		Debug:  f.Config.Debug,
	}

	var deviceCmd = &cobra.Command{
		Use:   "device",
		Short: "Perform actions on registered devices and their tokens",
		Long:  `The deivce command allows you to renew or revoke tokens used in the Collective.`,
	}

	deviceCmd.PersistentFlags().BoolVar(&opts.useJSON, "json", false, "Display in JSON format")

	deviceCmd.AddCommand(NewDeviceRevokeCmd(opts))
	deviceCmd.AddCommand(NewDeviceListCmd(opts))

	return deviceCmd
}
