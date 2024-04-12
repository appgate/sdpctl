package cmd

import (
	"context"
	"io"
	"text/template"

	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type adminMessageOpts struct {
	config  *configuration.Config
	factory *factory.Factory
	out     io.Writer
	json    bool
}

// NewAdminMessageCmd return a new admin message command
func NewAdminMessageCmd(f *factory.Factory) *cobra.Command {
	opts := adminMessageOpts{
		config:  f.Config,
		out:     f.IOOutWriter,
		factory: f,
	}
	cmd := &cobra.Command{
		Use:   "admin-messages",
		Short: docs.AdminMessagesRootDoc.Short,
		RunE: func(c *cobra.Command, args []string) error {
			return adminMessagesRun(&opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")

	return cmd
}

const messageTemplate = `Messages:
{{-  range . }}
{{ .GetLevel }} from {{ range .GetSources }}{{ . }}{{ end }}
{{ .GetCreated }}{{if gt .GetCount 1.0}} - {{ .GetCount }} occurrences{{end}}
{{ .GetMessage }}
{{ end }}
`

func adminMessagesRun(opts *adminMessageOpts) error {
	cfg := opts.config
	client, err := opts.factory.APIClient(cfg)
	if err != nil {
		return err
	}
	token, err := cfg.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	adminMessagesAPI := client.AdminMessagesApi
	ctx := context.Background()
	list, response, err := adminMessagesAPI.AdminMessagesSummarizeGet(ctx).Authorization(token).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if opts.json {
		return util.PrintJSON(opts.out, list)
	}

	t := template.Must(template.New("").Parse(messageTemplate))
	if err := t.Execute(opts.out, list.GetData()); err != nil {
		return err
	}
	return nil
}
