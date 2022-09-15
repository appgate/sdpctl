package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/util"
)

func GetStorageDirectory() string {
	p, err := Read()
	if err != nil {
		return filesystem.ConfigDir()
	}
	if p.CurrentExists() {
		return *p.Current
	}
	return filesystem.ConfigDir()
}

func FilePath() string {
	return filepath.Join(filesystem.ConfigDir(), "profiles.json")
}

func Directories() string {
	return filepath.Join(filesystem.ConfigDir(), "profiles")
}

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

func DirectoryExists() bool {
	if ok, err := util.FileExists(Directories()); err == nil && ok {
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

func CreateDirectories() error {
	return os.Mkdir(Directories(), os.ModePerm)
}

func Read() (*Profiles, error) {
	content, err := os.ReadFile(FilePath())
	if err != nil {
		return nil, fmt.Errorf("Can't read profiles: %s %s\n", FilePath(), err)
	}

	var profiles Profiles
	if err := json.Unmarshal(content, &profiles); err != nil {
		return nil, fmt.Errorf("%s file is corrupt: %s \n", FilePath(), err)
	}
	return &profiles, nil
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
