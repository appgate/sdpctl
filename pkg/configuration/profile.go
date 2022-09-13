package configuration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/util"
)

var (
	ProfileFilePath = filepath.Join(filesystem.ConfigDir(), "profiles.json")
	ProfileDirecty  = filepath.Join(filesystem.ConfigDir(), "profiles")
)

type Profiles struct {
	Current *string   `json:"current,omitempty"`
	List    []Profile `json:"list,omitempty"`
}

type Profile struct {
	Directory string `json:"directory"`
	Name      string `json:"name"`
}

func (p *Profiles) CurrentExists() bool {
	if p.Current != nil {
		if ok, err := util.FileExists(*p.Current); err == nil && ok {
			return true
		}
	}
	return false
}
func (p *Profiles) CurrentConfigExists() bool {
	if p.Current != nil {
		currentConfig := filepath.Join(*p.Current, "config.json")
		if ok, err := util.FileExists(currentConfig); err == nil && ok {
			return true
		}
	}
	return false
}

var ErrNoCurrentProfile = errors.New("no current profile is set, run 'sdpctl collective set'")

func (p *Profiles) CurrentProfile() (*Profile, error) {
	if p.Current == nil {
		return nil, ErrNoCurrentProfile
	}
	for _, profile := range p.List {
		if *p.Current == profile.Directory {
			return &profile, nil
		}
	}
	return nil, errors.New("could not get current profile")
}

func (p *Profile) CurrentConfig() (*Config, error) {
	currentConfig := filepath.Join(p.Directory, "config.json")
	if ok, err := util.FileExists(currentConfig); err == nil && ok {
		content, err := os.ReadFile(currentConfig)
		if err != nil {
			return nil, fmt.Errorf("Can not read current: %s %s\n", currentConfig, err)
		}

		var config Config
		if err := json.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("%s file is corrupt: %s \n", currentConfig, err)
		}
		return &config, nil
	}
	return nil, errors.New("current profile is not configured")
}

func ProfileDirectoryExists() bool {
	if ok, err := util.FileExists(ProfileDirecty); err == nil && ok {
		return true
	}
	return false
}

func ProfileFileExists() bool {
	if ok, err := util.FileExists(ProfileFilePath); err == nil && ok {
		return true
	}
	return false
}

func CreateProfileDirectory() error {
	return os.Mkdir(ProfileDirecty, os.ModePerm)
}

func ReadProfiles() (*Profiles, error) {
	content, err := os.ReadFile(ProfileFilePath)
	if err != nil {
		return nil, fmt.Errorf("Can't read profiles: %s %s\n", ProfileFilePath, err)
	}

	var profiles Profiles
	if err := json.Unmarshal(content, &profiles); err != nil {
		return nil, fmt.Errorf("%s file is corrupt: %s \n", ProfileFilePath, err)
	}
	return &profiles, nil
}

func WriteProfiles(p *Profiles) error {
	file, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(ProfileFilePath, file, 0644); err != nil {
		return err
	}
	return nil
}
