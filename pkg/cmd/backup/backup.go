package backup

import (
	"fmt"
	"net/url"
	"os"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/appliance"
	log "github.com/sirupsen/logrus"
)

func Prepare(c *config.Config, d string) error {
	log.Info("Preparing backup...")

	if appliance.IsOnAppliance() {
		return fmt.Errorf("This should not be executed on an appliance")
	}

	if _, err := os.Stat(d); os.IsNotExist(err) {
		if err := os.MkdirAll(d, 0700); err != nil {
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

func Perform(c *config.Config) error {
	log.Infof("Performing backup of controller at url %s", c.Url)

	return nil
}
