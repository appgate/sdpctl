package cmd

import (
	"fmt"
	"os"

	"github.com/appgate/appgatectl/internal/config"
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
	debug         bool
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
		PreRun:  preRunFunc,
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

	configureCmd := NewCmdConfigure(f)
	configureCmd.AddCommand(cfgcmd.NewLoginCmd(f))
	rootCmd.AddCommand(configureCmd)

	return rootCmd
}

func Execute() {
	root := NewCmdRoot()
	if err := root.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func preRunFunc(cmd *cobra.Command, args []string) {
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}
