package appliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/appgate/appgatectl/internal"
	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
)

var (
	DefaultBackupDestination = "$HOME/appgate/appgate_backup_yyyymmdd_hhMMss"
)

type BackupOpts struct {
	Config             *config.Config
	Appliance          func(*config.Config) (*Appliance, error)
	Out                io.Writer
	Destination        string
	NotifyURL          string
	Include            []string
	AllFlag            bool
	AllControllersFlag bool
}

type backupHTTPResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func PrepareBackup(opts *BackupOpts) error {
	log.Info("Preparing backup...")
	log.Debug(opts.Destination)

	if IsOnAppliance() {
		return fmt.Errorf("This should not be executed on an appliance")
	}

	if opts.Destination == DefaultBackupDestination {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		opts.Destination = filepath.FromSlash(fmt.Sprintf("%s/appgate/backup", homedir))
	}

	if err := os.MkdirAll(opts.Destination, 0700); err != nil {
		return err
	}

	return nil
}

func PerformBackup(opts *BackupOpts) error {
	ctx := context.Background()
	aud := internal.InSlice("audit", opts.Include)
	logs := internal.InSlice("logs", opts.Include)

	iObj := *openapi.NewInlineObject()
	iObj.Audit = &aud
	iObj.Logs = &logs

	if opts.Config.Version >= 16 {
		// introduced in v16
		iObj.NotifyUrl = &opts.NotifyURL
	}

	appliance, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}

	appliances, err := appliance.GetAll(ctx)
	if err != nil {
		return err
	}

	var toUpgrade []openapi.Appliance

	host, err := opts.Config.GetHost()
	if err != nil {
		return err
	}
	primaryController, err := FindPrimaryController(appliances, host)
	if err != nil {
		log.Debug(err)
		return fmt.Errorf("Failed to find primary controller")
	}
	toUpgrade = append(toUpgrade, *primaryController)

	if opts.AllFlag {
		toUpgrade = appliances
	}

	for _, a := range toUpgrade {
		log.Infof("Starting backup on %s...", a.Name)
		log.Debug(a.GetId())
		appliance.APIClient.GetConfig().AddDefaultHeader("Accept", "application/vnd.appgate.peer-v15+json")
		run := appliance.APIClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, a.Id).Authorization(appliance.Token).InlineObject(iObj)
		res, httpresponse, err := run.Execute()
		if err != nil {
			respBody := backupHTTPResponse{}
			decodeErr := json.NewDecoder(httpresponse.Body).Decode(&respBody)
			if decodeErr != nil {
				return decodeErr
			}
			log.Debug(respBody.Message)
			log.Debug(err)
			return fmt.Errorf("%s\nMessage: %s", err, respBody.Message)
		}
		backupID := res.GetId()

		var status string
		for status != "done" {
			status, err = getBackupState(ctx, appliance.APIClient, appliance.Token, a.Id, backupID)
			if err != nil {
				return err
			}
			time.Sleep(1 * time.Second)
		}

		appliance.APIClient.GetConfig().AddDefaultHeader("Accept", "application/vnd.appgate.peer-v15+gpg")
		file, inlineRes, err := appliance.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdGet(ctx, a.Id, backupID).Authorization(appliance.Token).Execute()
		if err != nil {
			log.Debug(err)
			log.Debug(inlineRes)
			return err
		}
		defer file.Close()
		dst, err := os.Create(fmt.Sprintf("%s/appgate_backup_%s_%s.bkp", opts.Destination, a.Name, time.Now().Format("20060102_150405")))
		if err != nil {
			return err
		}
		defer dst.Close()

		log.Debug("Downloading file...")
		_, err = io.Copy(dst, file)
		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("wrote backup file to '%s/%s'", opts.Destination, dst.Name()))
	}

	return nil
}

func getBackupState(ctx context.Context, client *openapi.APIClient, token string, aID string, bID string) (string, error) {
	res, _, err := client.ApplianceBackupApi.AppliancesIdBackupBackupIdStatusGet(ctx, aID, bID).Authorization(token).Execute()
	if err != nil {
		log.Debug(err)
		return "", err
	}
	log.Debug(*res.Status)

	return *res.Status, nil
}
