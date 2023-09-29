package profile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var validProfileName = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)

// NewAddCmd return a new profile add command
func NewAddCmd(opts *commandOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "add [<name>]",
		Short: docs.ProfileAddDoc.Short,
		Long:  docs.ProfileAddDoc.Long,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("requires one argument [profile-name]")
			}
			if args[0] == defaultProfileName {
				return fmt.Errorf("profile name %q is a reserved name, try another name", defaultProfileName)
			}
			if !validProfileName.MatchString(args[0]) {
				return fmt.Errorf("%q is not a valid profile name", args[0])
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return addRun(c, args, opts)
		},
	}
}

// defaultProfileName is the profile name if already have a config populated
// before we run the command
var defaultProfileName string = "default"

func addRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !profiles.FileExists() {
		profileFile, err := os.Create(profiles.FilePath())
		if err != nil {
			return fmt.Errorf("unable to create profiles directory %w", err)
		}
		defer profileFile.Close()
		if _, err := profileFile.WriteString("{}"); err != nil {
			return err
		}
	}
	if !profiles.ConfigDirectoryExists() {
		if err := profiles.CreateConfigDirectory(); err != nil {
			return fmt.Errorf("failed to create profile directory %w", err)
		}
	}
	if !profiles.LogDirectoryExists() {
		if err := profiles.CreateLogDirectory(); err != nil {
			return fmt.Errorf("failed to create profile log directory %w", err)
		}
	}

	p, err := profiles.Read()
	if err != nil {
		return err
	}

	if len(p.List) == 0 {
		// if profile.List is empty and we have a existing config.json in %SDPCTL_CONFIG_DIR
		// migrate the existing profile to a profile before adding a new one
		rootConfig := filepath.Join(filesystem.ConfigDir(), "config.json")
		if ok, err := util.FileExists(rootConfig); err == nil && ok {
			// move to profile default if it has a address in the config
			config, err := readConfig(rootConfig)
			if err != nil {
				return err
			}
			if h, err := config.GetHost(); len(h) > 0 && err == nil {
				cfg, logDir := profiles.Directories()
				directory := filepath.Join(cfg, defaultProfileName)
				if err := os.Mkdir(directory, os.ModePerm); err != nil {
					return fmt.Errorf("could not create new default config profile directory %w", err)
				}
				files, err := moveDefaultConfigFiles(filesystem.ConfigDir(), directory)
				if err != nil {
					return err
				}
				oldDefaultLogPath := filepath.Join(filesystem.DataDir(), "sdpctl.log")
				defaultLogPath := filepath.Join(logDir, "sdpctl.log")
				newLogPath := filepath.Join(logDir, "default.log")
				if ok, err := util.FileExists(oldDefaultLogPath); err == nil && ok {
					if err := os.Rename(oldDefaultLogPath, newLogPath); err != nil {
						return err
					}
				}
				if ok, err := util.FileExists(defaultLogPath); err == nil && ok {
					if err := os.Rename(defaultLogPath, newLogPath); err != nil {
						return err
					}
				}
				if err := updatePemPath(directory, files); err != nil {
					fmt.Fprintf(opts.Out, "could not update PEM file path for default config %s\n", err)
				}
				p.List = append(p.List, profiles.Profile{
					Name:      defaultProfileName,
					LogPath:   newLogPath,
					Directory: directory,
				})
				p.Current = &defaultProfileName
			}
		}
	}
	name := args[0]
	cfg, logs := profiles.Directories()
	directory := filepath.Join(cfg, name)
	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		return fmt.Errorf("profile already exists with the name %s", name)
	}
	if ok, err := util.FileExists(logs); err == nil && !ok {
		if err := os.Mkdir(logs, os.ModePerm); err != nil {
			return fmt.Errorf("failed creating log directory for %s", name)
		}
	}

	p.List = append(p.List, profiles.Profile{
		Name:      name,
		Directory: directory,
		LogPath:   filepath.Join(logs, name+".log"),
	})

	if err := profiles.Write(p); err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "Created profile %s, run 'sdpctl profile list' to see all available profiles\n", name)
	fmt.Fprintf(opts.Out, "run 'sdpctl profile set %s' to select the new profile\n", name)
	return nil
}

// updatePemPath if we have a existing pem file in the default config, update the pem_filepath
// to respect the new directory tree
// TODO: Deprecated. remove when suited
func updatePemPath(targetDir string, files []string) error {
	var (
		pemFilePath    string
		pemFileName    string
		ConfigFilePath string
	)
	for _, file := range files {
		f := filepath.Base(file)
		if strings.HasSuffix(f, ".pem") {
			pemFilePath = file
			pemFileName = f
		}
		if f == "config.json" {
			ConfigFilePath = file
		}
	}

	if ok, err := util.FileExists(ConfigFilePath); err == nil && ok {
		config, err := readConfig(ConfigFilePath)
		if err != nil {
			return err
		}
		if len(config.PemFilePath) > 0 && filepath.Base(config.PemFilePath) == pemFileName {
			config.PemFilePath = pemFilePath
			viper.Set("pem_filepath", pemFilePath)
			viper.SetConfigFile(filepath.Join(targetDir, "config.json"))
			if err := viper.WriteConfig(); err != nil {
				return err
			}
		}
	}
	return nil
}

func moveDefaultConfigFiles(root, target string) ([]string, error) {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	type result struct {
		path string
		info os.FileInfo
	}

	files := make(chan result)
	excludes := []string{"default", "(profiles(.json)?)", `(\w+\.log)`}
	g.Go(func() error {
		defer close(files)
		return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if regexp.MustCompile(strings.Join(excludes, "|")).Match([]byte(path)) {
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			select {
			case files <- result{path, info}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	})

	c := make(chan string)
	for i := 0; i < 3; i++ {
		g.Go(func() error {
			for file := range files {
				new := filepath.Join(target, file.info.Name())
				if err := os.Rename(file.path, new); err != nil {
					return err
				}
				select {
				case c <- new:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}
	go func() {
		g.Wait()
		close(c)
	}()

	r := make([]string, 0)
	for f := range c {
		r = append(r, f)
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return r, nil
}
