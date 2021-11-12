package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	appliancecmd "github.com/appgate/appgatectl/cmd/appliance"
	cfgcmd "github.com/appgate/appgatectl/cmd/configure"
	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version       string = "dev"
	cfgFile       string
	commit        string
	buildDate     string
	versionOutput string = fmt.Sprintf(`%s
commit: %s
build date: %s`, version, commit, buildDate)
)

func initConfig() {
	dir := configuration.ConfigDir()
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.Mkdir(dir, os.ModePerm)
			if err != nil {
				fmt.Printf("Can't create config dir: %s %s\n", dir, err)
				os.Exit(1)
			}
		}
		viper.AddConfigPath(dir)
		viper.SetEnvPrefix("APPGATECTL")
		viper.AutomaticEnv()
		viper.SetConfigType("json")
		viper.SafeWriteConfig()
		viper.SetConfigName("config")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Its OK if we cant the file, fallback to arguments and/or environment variables
			// or configure it with appgatectl configure
		} else {
			fmt.Printf("can't find config; run appgatectl configure %s %s\n", dir, err)
			os.Exit(1)
		}
	}
}

func NewCmdRoot() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:     "appgatectl [COMMAND]",
		Short:   "appgatectl is a command line tool to control and handle Appgate SDP using the CLI",
		Version: versionOutput,
		Aliases: []string{
			"agctl",
			"ag",
		},
	}

	cobra.OnInitialize(initConfig)
	cfg := &configuration.Config{}

	viper.SetDefault("debug", false)
	viper.SetDefault("provider", "local")

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
	rootCmd.PersistentFlags().BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")
	initConfig()

	viper.Unmarshal(cfg)
	f := factory.New(version, cfg)

	rootCmd.AddCommand(cfgcmd.NewCmdConfigure(f))
	rootCmd.AddCommand(appliancecmd.NewApplianceCmd(f))

	rootCmd.PersistentPreRunE = rootPersistentPreRunEFunc(f, cfg)

	return rootCmd
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
	if err := root.Execute(); err != nil {
		if errors.Is(err, ErrExitAuth) {
			return exitAuth
		}
		if errors.Is(err, appliance.ErrExecutionCanceledByUser) {
			return exitCancel
		}
		return exitError
	}
	return exitOK
}

func rootPersistentPreRunEFunc(f *factory.Factory, cfg *configuration.Config) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		logLevel := strings.ToLower(util.Getenv("APPGATECTL_LOG_LEVEL", "info"))

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
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			PadLevelText:    true,
		})
		log.SetOutput(f.IOOutWriter)

		// require that the user is authenticated before running most commands
		if configuration.IsAuthCheckEnabled(cmd) && !cfg.CheckAuth() {
			fmt.Fprintln(os.Stderr, "appgatectl err")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "To authenticate, please run `appgatectl configure login`.")
			return ErrExitAuth
		}

		return nil
	}
}
