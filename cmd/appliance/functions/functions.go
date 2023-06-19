package functions

import (
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

var (
	ValidFuncs []string = []string{
		appliancepkg.FunctionLogServer,
	}
	UnavailableFuncs []string = []string{
		appliancepkg.FunctionController,
		appliancepkg.FunctionGateway,
		appliancepkg.FunctionLogForwarder,
		appliancepkg.FunctionConnector,
		appliancepkg.FunctionPortal,
	}
)

func NewApplianceFunctionsCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use: "functions",
	}

	cmd.AddCommand(NewApplianceFunctionsDownloadCmd(f))

	return cmd
}
