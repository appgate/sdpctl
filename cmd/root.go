package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/appgate/appgatectl/internal"
	"github.com/appgate/appgatectl/internal/config"
	appliancecmd "github.com/appgate/appgatectl/pkg/cmd/appliance"
	cfgcmd "github.com/appgate/appgatectl/pkg/cmd/config"
	"github.com/appgate/appgatectl/pkg/cmd/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DefaultAPIVersion is based on the latest peer api version.
const DefaultAPIVersion = 16

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
	dir := config.ConfigDir()
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
			fmt.Printf("cant find config; run appgatectl configure %s %s\n", dir, err)
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
	cfg := &config.Config{}

	viper.SetDefault("debug", false)
	viper.SetDefault("provider", "local")

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
	rootCmd.PersistentFlags().BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")
	initConfig()

	viper.Unmarshal(cfg)
	f := factory.New(version, cfg)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logLevel := strings.ToLower(internal.Getenv("APPGATECTL_LOG_LEVEL", "info"))

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

		return nil
	}

	configureCmd := NewCmdConfigure(f)
	configureCmd.AddCommand(cfgcmd.NewLoginCmd(f))
	rootCmd.AddCommand(configureCmd)

	applianceCmd := NewApplianceCmd(f)
	applianceUpgradeCommand := appliancecmd.NewUpgradeCmd(f)
	applianceUpgradeCommand.AddCommand(appliancecmd.NewUpgradeStatusCmd(f))
	applianceUpgradeCommand.AddCommand(appliancecmd.NewPrepareUpgradeCmd(f))

	applianceCmd.AddCommand(applianceUpgradeCommand)
	rootCmd.AddCommand(applianceCmd)
	return rootCmd
}

func Execute() {
	root := NewCmdRoot()
	if err := root.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
