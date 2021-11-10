package appliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
)

var (
	DefaultBackupDestination = "$HOME/appgate/appgate_backup_yyyymmdd_hhMMss"
)

type BackupOpts struct {
	Config      *config.Config
	APIClient   func(*config.Config) (*openapi.APIClient, error)
	Out         io.Writer
	Destination string
	Audit       bool
	Logs        bool
	NotifyURL   string
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

	u, err := url.Parse(opts.Config.URL)
	if err != nil {
		return fmt.Errorf("Failed to parse controller url:\n\t%s", err)
	}
	log.Debug("Controller URL: ", u)

	return nil
}

func PerformBackup(opts *BackupOpts) error {
	client, err := opts.APIClient(opts.Config)
	if err != nil {
		return err
	}
	ctx := context.Background()
	token := opts.Config.GetBearTokenHeaderValue()
	iObj := *openapi.NewInlineObject()
	iObj.Audit = &opts.Audit
	iObj.Logs = &opts.Logs
	if opts.Config.Version >= 16 {
		// introduced in v16
		iObj.NotifyUrl = &opts.NotifyURL
	}
	appliances, err := GetAllAppliances(ctx, client, token)
	if err != nil {
		return err
	}
	for _, a := range appliances {
		fmt.Printf("Starting backup on %s...\n", a.Name)
		log.Debug(a.GetId())
        client.GetConfig().AddDefaultHeader("Accept", "application/vnd.appgate.peer-v15+json")
		run := client.ApplianceBackupApi.AppliancesIdBackupPost(ctx, a.Id).Authorization(token).InlineObject(iObj)
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
			status, err = getBackupState(ctx, client, token, a.Id, backupID)
			if err != nil {
				return err
			}
			time.Sleep(1 * time.Second)
		}

        client.GetConfig().AddDefaultHeader("Accept", "application/vnd.appgate.peer-v15+gpg")
		file, inlineRes, err := client.ApplianceBackupApi.AppliancesIdBackupBackupIdGet(ctx, a.Id, backupID).Authorization(token).Execute()
        if err != nil {
            log.Debug(err)
            log.Debug(inlineRes)
            return err
        }
        defer file.Close()
        dst, err := os.Create(fmt.Sprintf("%s/appgate_backup_%s_%s.bkp", opts.Destination, backupID, time.Now().Format("20060102_150405")))
        if err != nil {
            return err
        }
        defer dst.Close()

        log.Debug("Downloading file...")
        _ , err = io.Copy(dst, file)
        if err != nil {
            return err
        }
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
