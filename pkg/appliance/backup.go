package appliance

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/appgate/appgatectl/internal/config"
	log "github.com/sirupsen/logrus"
)

var (
	DefaultBackupDestination = "$HOME/appgate/appgate_backup_yyyymmdd_hhMMss"
)

func PrepareBackup(c *config.Config, destination string) error {
	log.Info("Preparing backup...")
	log.Debug(destination)

	if IsOnAppliance() {
		return fmt.Errorf("This should not be executed on an appliance")
	}

	if destination == DefaultBackupDestination {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		destination = filepath.FromSlash(fmt.Sprintf("%s/appgate/appgate_backup_%s", homedir, time.Now().Format("20060102_150405")))
	}

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		if err := os.MkdirAll(destination, 0700); err != nil {
			return fmt.Errorf("Failed to create destination directory:\n\t%s", err)
		}
	}

	u, err := url.Parse(c.Url)
	if err != nil {
		return fmt.Errorf("Failed to parse controller url:\n\t%s", err)
	}
	log.Debug("Controller URL: ", u)

	return nil
}

func PerformBackup(c *config.Config) error {
	log.Infof("Performing backup of controller at url %s", c.Url)

	return nil
}
