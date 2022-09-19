package serviceusers

import (
	"io"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/serviceusers"
	"github.com/spf13/cobra"
)

type ServiceUsersOptions struct {
	Config *configuration.Config
	API    func(c *configuration.Config) (*serviceusers.ServiceUsersAPI, error)
	Out    io.Writer
	JSON   bool
}

// ServiceUserDTO represents a service users options which are modifiable by the user
type ServiceUserDTO struct {
	Name     string            `json:"name,omitempty"`
	Password string            `json:"password,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
	Notes    string            `json:"notes,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}
type ServiceUserArrayDTO []ServiceUserDTO

var InvalidUUIDError = "argument is not a valid UUID"

func NewServiceUsersCMD(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service-users",
		Short:   docs.ServiceUsersRoot.Short,
		Long:    docs.ServiceUsersRoot.Long,
		Example: docs.ServiceUsersRoot.ExampleString(),
		Aliases: []string{"service-user", "su"},
	}

	cmd.PersistentFlags().Bool("json", false, "output in json format")
	cmd.AddCommand(NewServiceUsersListCMD(f))
	cmd.AddCommand(NewServiceUsersCreateCMD(f))
	cmd.AddCommand(NewServiceUsersGetCMD(f))
	cmd.AddCommand(NewServiceUsersDeleteCMD(f))
	cmd.AddCommand(NewServiceUsersUpdateCMD(f))

	return cmd
}
