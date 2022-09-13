package collective

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var validCollectiveName = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)

// NewAddCmd return a new collective add command
func NewAddCmd(opts *commandOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "",
		Long:  "",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("requires one argument [profile-name]")
			}
			if !validCollectiveName.MatchString(args[0]) {
				return fmt.Errorf("%q is not a valid collective name", args[0])
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return addRun(c, args, opts)
		},
	}
}

// defaultCollectiveName is the profile name if already have a config populated
// before we run the command
const defaultCollectiveName string = "default"

func addRun(cmd *cobra.Command, args []string, opts *commandOpts) error {
	if !configuration.ProfileFileExists() {
		profileFile, err := os.Create(configuration.ProfileFilePath)
		if err != nil {
			return fmt.Errorf("unable to create profiles directory %w", err)
		}
		defer profileFile.Close()
		if _, err := profileFile.WriteString("{}"); err != nil {
			return err
		}
	}
	if !configuration.ProfileDirectoryExists() {
		if err := configuration.CreateProfileDirectory(); err != nil {
			return fmt.Errorf("could not create profile directory %w", err)
		}
	}

	profiles, err := configuration.ReadProfiles()
	if err != nil {
		return err
	}

	if len(profiles.List) == 0 {
		// if profile.List is empty and we have a existing config.json in %SDPCTL_CONFIG_DIR
		// migrate the existing profile to a profile before adding a new one
		rootConfig := filepath.Join(filesystem.ConfigDir(), "config.json")
		if ok, err := util.FileExists(rootConfig); err == nil && ok {
			// move to profile default
			directory := filepath.Join(configuration.ProfileDirecty, defaultCollectiveName)
			if err := os.Mkdir(directory, os.ModePerm); err != nil {
				return fmt.Errorf("could not create new default config profile directory %w", err)
			}
			files, err := moveDefaultConfigFiles(filesystem.ConfigDir(), directory)
			if err != nil {
				return err
			}
			if err := updatePemPath(directory, files); err != nil {
				fmt.Fprintf(opts.Out, "could not update PEM file path for default config %s\n", err)
			}
			profiles.List = append(profiles.List, configuration.Profile{
				Name:      defaultCollectiveName,
				Directory: directory,
			})
			profiles.Current = &directory
		}
	}
	name := args[0]
	directory := filepath.Join(configuration.ProfileDirecty, name)
	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		return fmt.Errorf("profile already exists with the name %s", name)
	}

	profiles.List = append(profiles.List, configuration.Profile{
		Name:      name,
		Directory: directory,
	})

	if err := configuration.WriteProfiles(profiles); err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "Created profile %s, run 'sdpctl collective list' to see all available profiles\n", name)
	fmt.Fprintf(opts.Out, "run 'sdpctl collective set %s' to select the new collective profile\n", name)
	return nil
}

// updatePemPath if we have a existing pem file in the default config, update the pem_filepath
// to respect the new directory tree
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
		content, err := os.ReadFile(ConfigFilePath)
		if err != nil {
			return err
		}
		raw := make(map[string]interface{})
		if err := json.Unmarshal(content, &raw); err != nil {
			return err
		}
		var config configuration.Config
		if err := mapstructure.Decode(raw, &config); err != nil {
			return fmt.Errorf("%s file is corrupt: %s \n", ConfigFilePath, err)
		}
		currentPemName := filepath.Base(config.PemFilePath)
		if currentPemName == pemFileName {
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
