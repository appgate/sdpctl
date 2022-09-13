package cmd

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/appgate/sdpctl/cmd/collective"
	"github.com/appgate/sdpctl/cmd/token"
	"github.com/hashicorp/go-multierror"

	appliancecmd "github.com/appgate/sdpctl/cmd/appliance"
	cfgcmd "github.com/appgate/sdpctl/cmd/configure"
	"github.com/appgate/sdpctl/pkg/auth"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version         string = "dev"
	commit          string
	buildDate       string
	longDescription string = `The official CLI tool for managing your Appgate SDP Collective.
With sdpctl, you can list, backup and upgrade your Appgate SDP Appliances with a single command.`
	versionOutput string = fmt.Sprintf(`%s
commit: %s
build date: %s`, version, commit, buildDate)
)

func initConfig() {
	dir := filesystem.ConfigDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			fmt.Printf("Can't create config dir: %s %s\n", dir, err)
			os.Exit(1)
		}
	}

	if configuration.ProfileFileExists() {
		content, err := os.ReadFile(configuration.ProfileFilePath)
		if err != nil {
			fmt.Printf("Can't read profiles: %s %s\n", configuration.ProfileFilePath, err)
			os.Exit(1)
		}

		var profiles configuration.Profiles
		if err := json.Unmarshal(content, &profiles); err != nil {
			fmt.Printf("%s file is corrupt: %s \n", configuration.ProfileFilePath, err)
			os.Exit(1)
		}
		if profiles.CurrentExists() {
			viper.AddConfigPath(*profiles.Current)
		}
	} else {
		// if we don't have any profiles
		// we will assume there is only one collective to respect
		// and we will default to base dir.
		viper.AddConfigPath(dir)
	}

	viper.SafeWriteConfig()
	viper.SetConfigName("config")
	viper.SetEnvPrefix("SDPCTL")
	viper.AutomaticEnv()
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Its OK if we can't the file, fallback to arguments and/or environment variables
			// or configure it with sdpctl configure
		} else {
			fmt.Printf("can't find config; run sdpctl configure %s %s\n", dir, err)
			os.Exit(1)
		}
	}
}

func NewCmdRoot() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "sdpctl",
		Short:             "sdpctl is a command line tool to control and handle Appgate SDP using the CLI",
		Long:              longDescription,
		Version:           versionOutput,
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
	}

	cobra.OnInitialize(initConfig)
	cfg := &configuration.Config{}

	viper.SetDefault("debug", false)
	pFlags := rootCmd.PersistentFlags()
	pFlags.BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")
	pFlags.IntVar(&cfg.Version, "api-version", cfg.Version, "peer API version override")
	pFlags.BoolVar(&cfg.Insecure, "no-verify", cfg.Insecure, "don't verify TLS on for this particular command, overriding settings from config file")
	pFlags.Bool("no-interactive", false, "suppress interactive prompt with auto accept")
	pFlags.Bool("ci-mode", false, "log to stderr instead of file and disable progress-bars")

	initConfig()
	BindEnvs(*cfg)
	viper.Unmarshal(cfg)

	f := factory.New(version, cfg)
	rootCmd.AddCommand(cfgcmd.NewCmdConfigure(f))
	rootCmd.AddCommand(appliancecmd.NewApplianceCmd(f))
	rootCmd.AddCommand(token.NewTokenCmd(f))
	rootCmd.AddCommand(NewCmdCompletion())
	rootCmd.AddCommand(NewHelpCmd(f))
	rootCmd.AddCommand(NewOpenCmd(f))
	rootCmd.AddCommand(collective.NewCollectiveCmd(f))
	rootCmd.AddCommand(generateCmd)
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

type exitCode int

var ErrExitAuth = errors.New("no authentication")

const (
	exitOK     exitCode = 0
	exitError  exitCode = 1
	exitCancel exitCode = 2
	exitAuth   exitCode = 4
)

func Execute() exitCode {
	root := NewCmdRoot()
	cmd, err := root.ExecuteC()
	if err != nil {
		var result *multierror.Error
		// Unwrap error and check if we have a nested multierr
		// if we do, we will make the errors flat for 1 level
		// otherwise, append error to new multierr list
		if we := errors.Unwrap(err); we != nil {
			if merr, ok := we.(*multierror.Error); ok {
				for _, e := range merr.Errors {
					result = multierror.Append(result, e)
				}
			} else {
				result = multierror.Append(result, err)
			}
		} else {
			result = multierror.Append(result, err)
		}

		// if error is DeadlineExceeded, add custom ErrCommandTimeout
		if errors.Is(err, context.DeadlineExceeded) {
			result = multierror.Append(result, cmdutil.ErrCommandTimeout)
		}

		// if we during any request get a SSL error, (un-trusted certificate) error, prompt the user to import the pem file.
		var sslErr x509.UnknownAuthorityError
		if errors.As(err, &sslErr) {
			result = multierror.Append(result, errors.New("Trust the certificate or import a PEM file using 'sdpctl configure --pem=<path/to/pem>'"))
		}

		// print all multierrors to stderr, then return correct exitcode based on error type
		fmt.Fprintln(os.Stderr, result.ErrorOrNil())

		if errors.Is(err, ErrExitAuth) {
			return exitAuth
		}
		if errors.Is(err, cmdutil.ErrExecutionCanceledByUser) {
			return exitCancel
		}
		// only show usage prompt if we get invalid args / flags
		errorString := err.Error()
		if strings.Contains(errorString, "arg(s)") || strings.Contains(errorString, "flag") || strings.Contains(errorString, "command") {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			return exitError
		}

		return exitError
	}
	return exitOK
}

// logOutput defaults to logfile in $XDG_DATA_HOME or $HOME/.local/share
// if no TTY is available, stdout will be used
func logOutput(cmd *cobra.Command, f *factory.Factory, cfg *configuration.Config) io.Writer {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
		PadLevelText:    true,
	})
	if v, err := cmd.Flags().GetBool("ci-mode"); err == nil && v {
		return f.StdErr
	}
	if !cmdutil.IsTTY(os.Stdout) && !cmdutil.IsTTY(os.Stderr) {
		return f.StdErr
	}

	name := filepath.Join(filesystem.DataDir(), "sdpctl.log")
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

		if !cmdutil.IsTTY(os.Stdout) {
			if err := cmd.Flags().Set("no-interactive", "true"); err != nil {
				return err
			}
			if err := cmd.Flags().Set("ci-mode", "true"); err != nil {
				return err
			}
			log.Info("Output is not TTY. Using no-interactive and ci-mode")
		}

		if v, err := cmd.Flags().GetBool("ci-mode"); err == nil && v {
			f.SetSpinnerOutput(io.Discard)
		}
		if !cmdutil.IsTTY(os.Stdout) && !cmdutil.IsTTY(os.Stderr) {
			f.SetSpinnerOutput(io.Discard)
		}

		// If the token has expired, prompt the user for credentials if they are saved in the keychain
		if configuration.IsAuthCheckEnabled(cmd) {
			var result error
			// For certain sub-commands we want to make sure that we are using the
			// latest api version available (appliance upgrade prepare and complete)
			// we wont use this check for all commands, and fallback to the config value
			// so we can reduce number oh http requests to the controller.
			if configuration.NeedUpdatedAPIVersionConfig(cmd) {
				minMax, err := auth.GetMinMaxAPIVersion(f)
				if err == nil && minMax != nil {
					viper.Set("api_version", minMax.Max)
					f.Config.Version = int(minMax.Max)
					if err := viper.WriteConfig(); err != nil {
						fmt.Fprintf(f.StdErr, "[error] %s\n", err)
					}
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
			result = multierror.Append(result, errors.New("To authenticate, please run `sdpctl configure signin`."))
			result = multierror.Append(result, ErrExitAuth)
			return result
		}
		return nil
	}
}
