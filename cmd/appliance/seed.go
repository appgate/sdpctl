package appliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type seedOpts struct {
	Config    *configuration.Config
	Out       io.Writer
	In        io.ReadCloser
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug     bool
	CanPrompt bool
	json      bool
	// Seed config options
	applianceID        string
	password           string
	sshKey             string
	provideCloudSSHKey bool
	allowCustomization bool
	validityDays       int
	iso                bool
}

type authmethod int

const (
	cloud authmethod = iota
	publicKey
	passphrase
)

// NewSeedCmd return a new appliance seed command
func NewSeedCmd(f *factory.Factory) *cobra.Command {
	opts := seedOpts{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		In:        f.Stdin,
		CanPrompt: f.CanPrompt(),
	}
	var cmd = &cobra.Command{
		Use:     "seed",
		Short:   "",
		Long:    "",
		Example: "",
		Args: func(cmd *cobra.Command, args []string) error {
			a, err := opts.Appliance(opts.Config)
			if err != nil {
				return err
			}
			ctx := context.Background()
			filter := map[string]map[string]string{
				"include": {
					"activated": "false",
				},
			}
			switch len(args) {
			// no arguments will provider a interactive experience for the user, if TTY support it
			case 0:
				if !opts.CanPrompt {
					return errors.New("can't prompt, You need to provider all arguments")
				}
				applianceID, err := appliancepkg.PromptSelectAll(ctx, a, filter)
				if err != nil {
					return err
				}
				opts.applianceID = applianceID
				methods := []string{
					cloud:      "Use SSH key provided by the cloud instance",
					publicKey:  "Use SSH public key",
					passphrase: "Use Password",
				}
				qs := &survey.Select{
					PageSize: len(methods),
					Message:  "SSH Authentication Method:",
					Options:  methods,
				}
				selectedIndex := 0
				if err := prompt.SurveyAskOne(qs, &selectedIndex, survey.WithValidator(survey.Required)); err != nil {
					return err
				}
				switch selectedIndex {
				case int(cloud):
					opts.provideCloudSSHKey = true
				case int(publicKey):
					qs := &survey.Input{
						Message: "path to public ssh key:",
					}
					if err := prompt.SurveyAskOne(qs, &opts.sshKey); err != nil {
						return err
					}
				case int(passphrase):
					hasStdin := false
					stat, err := os.Stdin.Stat()
					if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
						hasStdin = true
					}
					password, err := prompt.GetPassphrase(opts.In, opts.CanPrompt, hasStdin, "Appliance's CZ user password:")
					if err != nil {
						return err
					}
					opts.password = password
				}

			case 1:
				if !util.IsUUID(args[0]) {
					return errors.New("expected appliance UUID")
				}
				opts.applianceID = args[0]
				opts.json = true
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return seedRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.provideCloudSSHKey, "provide-cloud-ssh-key", false, "Tells appliance to use the key generated by AWS or Azure")
	cmd.Flags().BoolVar(&opts.allowCustomization, "allow-customization", false, "Whether the Appliance should allow customizations or not")
	cmd.Flags().BoolVar(&opts.iso, "iso", false, "Export as ISO format")
	cmd.Flags().StringVarP(&opts.sshKey, "ssh-key", "", "", "filepath to public ssh key to be allowed")
	cmd.Flags().IntVarP(&opts.validityDays, "validity-days", "", 1, "How many days the seed should be valid for")

	return cmd
}

func seedRun(cmd *cobra.Command, args []string, opts *seedOpts) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	token, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	ctx := context.Background()
	appliance, err := a.Get(ctx, opts.applianceID)
	if err != nil {
		return err
	}
	if appliance.GetActivated() {
		return fmt.Errorf("appliance %s is already activated", appliance.GetName())
	}

	sshConfig := openapi.NewSSHConfig()

	if cfg.Version < 15 {
		sshConfig.AllowCustomization = nil
		sshConfig.ValidityDays = nil
	}
	sshConfig.AllowCustomization = openapi.PtrBool(opts.allowCustomization)
	sshConfig.ValidityDays = openapi.PtrFloat32(float32(opts.validityDays))

	if len(opts.sshKey) > 0 {
		pub, err := readFile(opts.sshKey)
		if err != nil {
			return err
		}
		sshConfig.SetSshKey(pub)
	} else if opts.provideCloudSSHKey {
		sshConfig.SetProvideCloudSSHKey(true)
	} else if len(opts.password) < 1 {
		hasStdin := false
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			hasStdin = true
		}
		password, err := prompt.GetPassphrase(opts.In, false, hasStdin, "")
		if err != nil {
			return errors.New("You need to provide passphrase through stdin or select --ssh-key=/path/to/key.pub or --provide-cloud-ssh-key")
		}
		sshConfig.SetPassword(password)
	} else {
		sshConfig.SetPassword(opts.password)
	}

	var (
		data []byte
		ext  string
	)
	if opts.iso {
		seed, response, err := a.APIClient.AppliancesApi.AppliancesIdExportIsoPost(ctx, appliance.GetId()).SSHConfig(*sshConfig).Authorization(token).Execute()
		if err != nil {
			return api.HTTPErrorResponse(response, err)
		}
		data = []byte(seed.GetIso())
		ext = "iso"
	} else {
		seed, response, err := a.APIClient.AppliancesApi.AppliancesIdExportPost(ctx, appliance.GetId()).SSHConfig(*sshConfig).Authorization(token).Execute()
		if err != nil {
			return api.HTTPErrorResponse(response, err)
		}
		encodedSeed, err := json.MarshalIndent(seed, "", " ")
		if err != nil {
			return fmt.Errorf("Could not parse json seed file: %w", err)
		}
		if opts.json {
			fmt.Fprintln(opts.Out, string(encodedSeed))
			return nil
		}
		data = encodedSeed
		ext = "json"
	}

	path, err := os.Getwd()
	if err != nil {
		return err
	}
	outputfile := filepath.Join(path, fmt.Sprintf("%s_seed.%s", appliance.GetId(), ext))
	if err := os.WriteFile(outputfile, data, 0644); err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "Saved appliance %s seed file to %s\n", appliance.GetName(), outputfile)

	return nil
}

func readFile(p string) (string, error) {
	dat, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}
