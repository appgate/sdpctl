package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/appgate/sdpctl/cmd/license"
	cmdprofile "github.com/appgate/sdpctl/cmd/profile"
	"github.com/appgate/sdpctl/cmd/token"
	"github.com/hashicorp/go-multierror"

	appliancecmd "github.com/appgate/sdpctl/cmd/appliance"
	cfgcmd "github.com/appgate/sdpctl/cmd/configure"
	"github.com/appgate/sdpctl/cmd/serviceusers"
	"github.com/appgate/sdpctl/pkg/auth"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	minSupportedVersion int    = 17 // e.g. appliance version 6.0.0
	version             string = "0.0.0-dev"
	commit              string
	buildDate           string
	longDescription     string = `The official CLI tool for managing your Collective.`
	versionOutput       string = fmt.Sprintf(`%s
commit: %s
build date: %s`, version, commit, buildDate)
	minAPIversionWarning string = `WARNING: You are running an unsupported API version on the appliance.
It is strongly adviced that you upgrade your appliance to a supported version before executing any sdpctl command. Executing sdpctl commands on an unsupported API version can have serious unforseen consequenses.
Minimum supported API version: %d
Currently using API version: %d

Consider upgrading to a supported version of the appliance using the upgrade script provided in the 'Utilities' section in the admin UI: %s

`
)

func initConfig(currentProfile *string) {
	dir := filesystem.ConfigDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			fmt.Printf("Can't create config dir: %s %s\n", dir, err)
			os.Exit(1)
		}
	}
	if profiles.FileExists() {
		p, err := profiles.Read()
		if err != nil {
			fmt.Printf("Can't read profiles: %s %s\n", profiles.FilePath(), err)
			os.Exit(1)
		}
		var selectedProfile string
		if currentProfile != nil && len(*currentProfile) > 0 {
			selectedProfile = *currentProfile
		} else if v := os.Getenv("SDPCTL_PROFILE"); len(v) > 0 {
			selectedProfile = v
		}
		if len(selectedProfile) > 0 {
			found := false
			for _, profile := range p.List {
				if selectedProfile == profile.Name {
					viper.AddConfigPath(profile.Directory)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Invalid profile name, got %s, available %s\n", selectedProfile, p.Available())
				os.Exit(1)
			}
		} else if p.CurrentExists() {
			viper.AddConfigPath(*p.Current)
		}
	} else {
		// if we don't have any profiles
		// we will assume there is only one Collective to respect
		// and we will default to base dir.
		viper.AddConfigPath(dir)
	}

	viper.SetConfigName("config")
	viper.SetEnvPrefix("SDPCTL")
	viper.AutomaticEnv()
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Its OK if we can't the file, fallback to arguments and/or environment variables
			// or configure it with sdpctl configure
		} else {
			fmt.Printf("Can't find config; run sdpctl configure %s %s\n", dir, err)
			os.Exit(1)
		}
	}
}

func NewCmdRoot(currentProfile *string) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "sdpctl",
		Short:             "sdpctl is a command line tool to manage Appgate SDP Collectives",
		Long:              longDescription,
		Version:           versionOutput,
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
	}

	cfg := &configuration.Config{}

	pFlags := rootCmd.PersistentFlags()
	pFlags.BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")
	pFlags.IntVar(&cfg.Version, "api-version", cfg.Version, "Peer API version override")
	pFlags.BoolVar(&cfg.Insecure, "no-verify", cfg.Insecure, "Don't verify TLS on for the given command, overriding settings from config file")
	pFlags.Bool("no-interactive", false, "Suppress interactive prompt with auto accept")
	pFlags.Bool("ci-mode", false, "Log to stderr instead of file and disable progress-bars")
	pFlags.String("events-path", "", "send logs to unix domain socket path")

	// hack this is just a dummy flag to show up in --help menu, the real flag is defined
	// in Execute() because we need to parse it first before the others to be able
	// to resolve factory.New
	var dummy string
	pFlags.StringVarP(&dummy, "profile", "p", "", "Profile configuration to use")

	initConfig(currentProfile)

	BindEnvs(*cfg)
	viper.Unmarshal(cfg)

	f := factory.New(version, cfg)
	rootCmd.AddCommand(cfgcmd.NewCmdConfigure(f))
	rootCmd.AddCommand(appliancecmd.NewApplianceCmd(f))
	rootCmd.AddCommand(token.NewTokenCmd(f))
	rootCmd.AddCommand(NewCmdCompletion())
	rootCmd.AddCommand(NewHelpCmd(f))
	rootCmd.AddCommand(NewOpenCmd(f))
	rootCmd.AddCommand(cmdprofile.NewProfileCmd(f))
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(serviceusers.NewServiceUsersCMD(f))
	rootCmd.AddCommand(license.NewLicenseCmd(f))
	rootCmd.SetUsageTemplate(UsageTemplate())
	rootCmd.SetHelpTemplate(HelpTemplate())
	rootCmd.PersistentPreRunE = rootPersistentPreRunEFunc(f, cfg)

	return rootCmd
}

// BindEnvs Consider env vars when unmarshalling
// https://github.com/spf13/viper/issues/188#issuecomment-399884438
func BindEnvs(iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			BindEnvs(v.Interface(), append(parts, tv)...)
		default:
			viper.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

func Execute() cmdutil.ExitCode {
	var currentProfile string
	rFlag := pflag.NewFlagSet("", pflag.ContinueOnError)
	rFlag.StringVarP(&currentProfile, "profile", "p", "", "")
	rFlag.Usage = func() {}
	rFlag.Parse(os.Args[1:])

	return cmdutil.ExecuteCommand(NewCmdRoot(&currentProfile))
}

// logOutput defaults to logfile in $XDG_DATA_HOME or $HOME/.local/share
// if no TTY is available, stdout will be used
func logOutput(cmd *cobra.Command, f *factory.Factory, cfg *configuration.Config) io.Writer {
	if v, err := cmd.Flags().GetString("events-path"); err == nil && len(v) > 0 {
		if err := util.AddSocketLogHook(v); err != nil {
			log.WithError(err).Error("failed to initialize events-path")
		}
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
		PadLevelText:    true,
	})
	noTTY := !cmdutil.IsTTY(os.Stdout) && !cmdutil.IsTTY(os.Stderr)
	if v, err := cmd.Flags().GetBool("ci-mode"); (err == nil && v) || noTTY {
		return f.StdErr
	}

	name := filepath.Join(profiles.GetStorageDirectory(), "sdpctl.log")
	file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return f.IOOutWriter
	}

	return file
}

func rootPersistentPreRunEFunc(f *factory.Factory, cfg *configuration.Config) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		logLevel := strings.ToLower(util.Getenv("SDPCTL_LOG_LEVEL", "info"))

		switch logLevel {
		case "panic":
			log.SetLevel(log.PanicLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		case "warn", "warning":
			log.SetLevel(log.WarnLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "trace":
			log.SetReportCaller(true)
			log.SetLevel(log.TraceLevel)
		default:
			log.SetLevel(log.ErrorLevel)
		}
		if cfg.Debug {
			log.SetLevel(log.DebugLevel)
		}

		log.SetOutput(logOutput(cmd, f, cfg))

		// log sdpctl version
		logFields := log.Fields{
			"SDPCTL_VERSION": version,
			"COMMAND":        cmd.CommandPath(),
		}
		if len(args) > 0 {
			logFields["ARGS"] = strings.Join(args, ",")
		}
		log.WithFields(logFields).Info()

		if !cmdutil.IsTTY(os.Stdout) {
			if err := cmd.Flags().Set("no-interactive", "true"); err != nil {
				return err
			}
			if err := cmd.Flags().Set("ci-mode", "true"); err != nil {
				return err
			}
			log.Debug("Output is not TTY. Using no-interactive and ci-mode")
		}

		if value, err := cmd.Flags().GetBool("no-interactive"); err == nil {
			f.DisablePrompt(value)
		}
		if v, err := cmd.Flags().GetBool("ci-mode"); err == nil && v {
			f.SetSpinnerOutput(io.Discard)
		}
		if !cmdutil.IsTTY(os.Stdout) && !cmdutil.IsTTY(os.Stderr) {
			f.SetSpinnerOutput(io.Discard)
		}

		// Check minimum supported version and print warning if the client is running an unsupported version
		// We check length of configured URL to not show warning when profile is unconfigured
		if cfg.Version < minSupportedVersion && len(cfg.URL) > 0 {
			utilitesURL, err := url.ParseRequestURI(cfg.URL)
			if err != nil {
				return err
			}
			utilitesURL.Path = `/ui/system/utilities`
			fmt.Fprintf(f.StdErr, minAPIversionWarning, minSupportedVersion, cfg.Version, utilitesURL)
		}

		// If the token has expired, prompt the user for credentials if they are saved in the keychain
		if configuration.IsAuthCheckEnabled(cmd) {
			var result error
			// For certain sub-commands we want to make sure that we are using the
			// latest api version available (appliance upgrade prepare and complete)
			// we wont use this check for all commands, and fallback to the config value
			// so we can reduce number oh http requests to the Controller.
			if configuration.NeedUpdatedAPIVersionConfig(cmd) {
				minMax, err := auth.GetMinMaxAPIVersion(f)
				if err == nil && minMax != nil {
					viper.Set("api_version", minMax.Max)
					f.Config.Version = int(minMax.Max)
				}
			}

			if err := auth.Signin(f); err != nil {
				result = multierror.Append(result, err)
				return result
			}
		}

		// require that the user is authenticated before running most commands
		if configuration.IsAuthCheckEnabled(cmd) && !cfg.CheckAuth() {
			var result error
			result = multierror.Append(result, errors.New("To authenticate, please run `sdpctl configure signin`"))
			result = multierror.Append(result, cmdutil.ErrExitAuth)
			return result
		}

		// Check minimum API version requirement for command
		if err := configuration.CheckMinAPIVersionRestriction(cmd, cfg.Version); err != nil {
			return err
		}

		// Check for new sdpctl version
		client, err := f.HTTPClient()
		if err != nil {
			return err
		}
		cfg, err = cfg.CheckForUpdate(f.StdErr, client, version)
		if err != nil {
			log.WithError(err).Info("version check result")
		}
		if err := viper.WriteConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				fmt.Fprintf(f.StdErr, "[error] %s\n", err)
			}
		}

		return nil
	}
}
