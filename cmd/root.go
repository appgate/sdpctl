package cmd

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/appgate/sdpctl/cmd/token"

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
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			fmt.Printf("Can't create config dir: %s %s\n", dir, err)
			os.Exit(1)
		}
	}
	viper.AddConfigPath(dir)
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
	viper.SetDefault("provider", "local")

	pFlags := rootCmd.PersistentFlags()
	pFlags.BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")
	pFlags.IntVar(&cfg.Version, "api-version", cfg.Version, "peer API version override")
	pFlags.BoolVar(&cfg.Insecure, "no-verify", cfg.Insecure, "don't verify TLS on for this particular command, overriding settings from config file")
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
	rootCmd.AddCommand(generateCmd)
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
		log.WithError(err).Error("Caught error")
		errorString := err.Error()
		fmt.Fprintln(os.Stderr, errorString)
		if errors.Is(err, ErrExitAuth) {
			return exitAuth
		}
		if errors.Is(err, cmdutil.ErrExecutionCanceledByUser) {
			return exitCancel
		}
		// only show usage prompt if we get invalid args / flags
		if strings.Contains(errorString, "arg(s)") || strings.Contains(errorString, "flag") || strings.Contains(errorString, "command") {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			return exitError
		}

		return exitError
	}
	return exitOK
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
			log.SetLevel(log.TraceLevel)
		default:
			log.SetLevel(log.ErrorLevel)
		}
		if cfg.Debug {
			log.SetLevel(log.DebugLevel)
		}

		fName := fmt.Sprintf("%s/sdpctl.log", filesystem.ConfigDir())
		file, err := os.OpenFile(fName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.SetFormatter(&log.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: "2006-01-02 15:04:05",
				PadLevelText:    true,
				ForceColors:     true,
			})
			log.Warn("Failed to open log file. Logging to stdout")
			log.SetOutput(f.IOOutWriter)
		} else {
			log.SetOutput(file)
		}

		if configuration.IsAuthCheckEnabled(cmd) && !cfg.CheckAuth() {
			if err := auth.Signin(f, false, false); err != nil {
				fmt.Fprintln(os.Stderr, "sdpctl authentication err")
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, err)
				return ErrExitAuth
			}
		}

		// require that the user is authenticated before running most commands
		if configuration.IsAuthCheckEnabled(cmd) && !cfg.CheckAuth() {
			fmt.Fprintln(os.Stderr, "sdpctl err")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "To authenticate, please run `sdpctl configure signin`.")
			return ErrExitAuth
		}

		return nil
	}
}
