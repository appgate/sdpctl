package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/appgate/sdpctl/pkg/auth"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type privilegeOption struct {
	config  *configuration.Config
	factory *factory.Factory
	out     io.Writer
	json    bool
}

// NewPrivilegesCmd return a new privileges command
func NewPrivilegesCmd(f *factory.Factory) *cobra.Command {
	opts := privilegeOption{
		config:  f.Config,
		out:     f.IOOutWriter,
		factory: f,
	}
	cmd := &cobra.Command{
		Use:   "privileges",
		Short: docs.PrivilegesDocs.Short,
		RunE: func(c *cobra.Command, args []string) error {
			return privilegeRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")

	return cmd
}

func privilegeRun(cmd *cobra.Command, args []string, opts *privilegeOption) error {
	cfg := opts.config
	client, err := opts.factory.APIClient(cfg)
	if err != nil {
		return err
	}
	token, err := cfg.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	authenticator := auth.NewAuth(client)

	response, err := authenticator.Authorization(context.Background(), token)
	if err != nil {
		return err
	}
	user := response.GetUser()
	privileges := user.GetPrivileges()
	if opts.json {
		return util.PrintJSON(opts.out, user)
	}
	fmt.Fprintf(opts.out, "\n%s have the following privileges\n\n", user.GetName())
	p := util.NewPrinter(opts.out, 4)
	p.AddHeader("target", "type", "scope")
	for _, privilege := range privileges {
		s := privilege.GetScope()
		p.AddLine(privilege.GetTarget(), privilege.GetType(), append(s.GetTags(), s.GetIds()...))
	}
	p.Print()
	return nil
}
