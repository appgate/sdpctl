package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
)

func GetCurrentProfile() string {
	p, err := Read()
	if err != nil {
		return "<no profile configured>"
	}
	if p.CurrentExists() {
		return *p.Current
	}
	return "default"
}

func GetConfigDirectory() string {
	p, err := Read()
	if err != nil {
		return filesystem.ConfigDir()
	}
	if p.CurrentExists() {
		return filepath.Join(filesystem.ConfigDir(), "profiles", *p.Current)
	}
	return filesystem.ConfigDir()
}

func GetConfigPath() string {
	defaultConfigPath := filepath.Join(filesystem.ConfigDir(), "config.json")
	p, err := Read()
	if err != nil {
		return defaultConfigPath
	}
	if p.CurrentExists() {
		return filepath.Join(filesystem.ConfigDir(), "profiles", *p.Current, "config.json")
	}
	return defaultConfigPath
}

func GetDataDirectory() string {
	p, err := Read()
	if err != nil {
		return filesystem.DataDir()
	}
	if p.CurrentExists() {
		return filepath.Join(filesystem.DataDir(), *p.Current)
	}
	return filesystem.DataDir()
}

func GetLogPath() string {
	defaultLogPath := filepath.Join(filesystem.DataDir(), "logs", "sdpctl.log")
	p, err := Read()
	if err != nil {
		return defaultLogPath
	}
	if p.CurrentExists() {
		return filepath.Join(filesystem.DataDir(), "logs", *p.Current+".log")
	}
	return defaultLogPath
}

func FilePath() string {
	return filepath.Join(filesystem.ConfigDir(), "profiles.json")
}

func Directories() (string, string) {
	return filepath.Join(filesystem.ConfigDir(), "profiles"), filepath.Join(filesystem.DataDir(), "logs")
}

type Profiles struct {
	Current *string   `json:"current,omitempty"`
	List    []Profile `json:"list,omitempty"`
}

type Profile struct {
	Directory string `json:"directory"`
	LogPath   string `json:"logs"`
	Name      string `json:"name"`
}

func (p *Profile) GetConfigurationPath() string {
	return filepath.Join(p.Directory, p.Name, "config.json")
}

func (p *Profile) GetLogPath() string {
	return p.LogPath
}

func (p *Profiles) CurrentExists() bool {
	if p.Current != nil {
		var profile *Profile
		for _, v := range p.List {
			if *p.Current == v.Name {
				profile = &v
			}
		}
		if profile != nil {
			if ok, err := util.FileExists(profile.Directory); err == nil && ok {
				return true
			}
		}
	}
	return false
}

func (p *Profiles) CreateDefaultProfile() (*Profile, error) {
	conf, logs := Directories()
	confDir := filepath.Join(conf, "default")
	logPath := filepath.Join(logs, "default.log")
	if ok, err := util.FileExists(confDir); err == nil && !ok {
		if err := os.MkdirAll(confDir, os.ModePerm); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if ok, err := util.FileExists(logs); err == nil && !ok {
		if err := os.MkdirAll(logs, os.ModePerm); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if ok, err := util.FileExists(logPath); err == nil && !ok {
		f, err := os.Create(logPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
	} else if err != nil {
		return nil, err
	}
	profile := Profile{
		Name:      "default",
		Directory: confDir,
		LogPath:   logPath,
	}
	return &profile, nil
}

func (p *Profiles) CurrentConfigExists() bool {
	if p.Current != nil {
		var profile *Profile
		for _, v := range p.List {
			if v.Name == *p.Current {
				profile = &v
			}
		}
		currentConfig := profile.GetConfigurationPath()
		if ok, err := util.FileExists(currentConfig); err == nil && ok {
			return true
		}
	}
	return false
}

func (p *Profiles) GetProfile(name string) (*Profile, error) {
	for _, v := range p.List {
		if v.Name == name {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("no profile found matching name %s", name)
}

var ErrNoCurrentProfile = errors.New("No current profile is set, run 'sdpctl profile set'")
var ErrNoProfileAvailable = errors.New("No profiles are available. run 'sdpctl profile set'")

func (p *Profiles) CurrentProfile() (*Profile, error) {
	if v := os.Getenv("SDPCTL_PROFILE"); len(v) > 0 {
		for _, profile := range p.List {
			if v == profile.Name {
				return &profile, nil
			}
		}
	}
	if p.Current == nil {
		return nil, ErrNoCurrentProfile
	}
	if len(p.List) <= 0 {
		return nil, ErrNoProfileAvailable
	}
	for _, profile := range p.List {
		if *p.Current == profile.Name {
			return &profile, nil
		}
	}
	return nil, errors.New("failed to get current profile")
}

func (p *Profiles) Available() []string {
	names := make([]string, 0)
	for _, profile := range p.List {
		names = append(names, profile.Name)
	}
	return names
}

func ConfigDirectoryExists() bool {
	cfg, _ := Directories()
	if ok, err := util.FileExists(cfg); err == nil && ok {
		return true
	}
	return false
}

func LogDirectoryExists() bool {
	_, dir := Directories()
	if ok, err := util.FileExists(dir); err == nil && ok {
		return true
	}
	return false
}

func FileExists() bool {
	if ok, err := util.FileExists(FilePath()); err == nil && ok {
		return true
	}
	return false
}

func CreateAllDirectories() error {
	var errs *multierror.Error
	cfg, log := Directories()
	if err := os.MkdirAll(cfg, os.ModePerm); err != nil {
		errs = multierror.Append(errs, err)
	}
	if err := os.MkdirAll(log, os.ModePerm); err != nil {
		errs = multierror.Append(errs, err)
	}
	return errs.ErrorOrNil()
}

func CreateConfigDirectory() error {
	cfg, _ := Directories()
	if err := os.MkdirAll(cfg, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func CreateLogDirectory() error {
	_, log := Directories()
	if err := os.MkdirAll(log, os.ModePerm); err != nil {
		return err
	}
	return nil
}

var ReadProfiles *Profiles

func Read() (*Profiles, error) {
	if ReadProfiles != nil {
		return ReadProfiles, nil
	}
	content, err := os.ReadFile(FilePath())
	if err != nil {
		return nil, fmt.Errorf("Can't read profiles: %s %s\n", FilePath(), err)
	}

	var profiles Profiles
	if err := json.Unmarshal(content, &profiles); err != nil {
		return nil, fmt.Errorf("%s file is corrupt: %s \n", FilePath(), err)
	}
	if profiles.Current != nil {
		c := filepath.Base(*profiles.Current)
		profiles.Current = &c
	}
	ReadProfiles = &profiles
	return ReadProfiles, nil
}

func Write(p *Profiles) error {
	file, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(FilePath(), file, 0644); err != nil {
		return err
	}
	return nil
}
