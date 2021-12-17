package appliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	DefaultBackupDestination = "$HOME/Downloads/appgate/backup"
)

type BackupOpts struct {
	Config             *configuration.Config
	Appliance          func(*configuration.Config) (*Appliance, error)
	Out                io.Writer
	Destination        string
	NotifyURL          string
	Include            []string
	AllFlag            bool
	AllControllersFlag bool
	Timeout            time.Duration
}

type backupHTTPResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func PrepareBackup(opts *BackupOpts) error {
	log.WithField("destination", opts.Destination).Info("Preparing backup...")

	if IsOnAppliance() {
		return fmt.Errorf("This should not be executed on an appliance")
	}

	if opts.Destination == DefaultBackupDestination {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		opts.Destination = filepath.FromSlash(fmt.Sprintf("%s/Downloads/appgate/backup", homedir))
	}

	if err := os.MkdirAll(opts.Destination, 0700); err != nil {
		return err
	}

	return nil
}

func PerformBackup(cmd *cobra.Command, opts *BackupOpts) (map[string]string, error) {
	backupIDs := make(map[string]string)
	ctx := context.Background()
	aud := util.InSlice("audit", opts.Include)
	logs := util.InSlice("logs", opts.Include)

	iObj := *openapi.NewInlineObject()
	iObj.Audit = &aud
	iObj.Logs = &logs

	if opts.Config.Version >= 16 && len(opts.NotifyURL) > 0 {
		// introduced in v16
		iObj.NotifyUrl = &opts.NotifyURL
	}

	app, err := opts.Appliance(opts.Config)
	if err != nil {
		return backupIDs, err
	}

	backupEnabled, err := backupEnabled(ctx, app.APIClient, opts.Config.GetBearTokenHeaderValue())
	if err != nil {
		return backupIDs, fmt.Errorf("Failed to determine backup option: %w", err)
	}
	if !backupEnabled {
		return backupIDs, fmt.Errorf("Backup API is disabled in the collective.")
	}

	filter := util.ParseFilteringFlags(cmd.Flags())
	if opts.AllControllersFlag {
		filter["filter"]["function"] = "controller"
	}
	appliances, err := app.List(ctx, filter)
	if err != nil {
		return backupIDs, err
	}

	var toBackup []openapi.Appliance

	host, err := opts.Config.GetHost()
	if err != nil {
		return backupIDs, err
	}
	primaryController, err := FindPrimaryController(appliances, host)
	if err != nil {
		log.Debug(err)
		return backupIDs, fmt.Errorf("Failed to find primary controller")
	}
	toBackup = append(toBackup, *primaryController)

	if opts.AllFlag {
		toBackup = appliances
	}

	// Filter offline appliances
	initialStats, _, err := app.Stats(ctx)
	if err != nil {
		return backupIDs, err
	}
	toBackup, offline, _ := FilterAvailable(toBackup, initialStats.GetData())

	if len(offline) > 0 {
		for _, v := range offline {
			log.WithField("appliance", v.GetName()).Info("Skipping appliance. Appliance is offline.")
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, a := range toBackup {
		appliance := a
		apiClient := app.APIClient
		g.Go(func() error {
			fields := log.Fields{"appliance": appliance.Name}
			log.WithFields(fields).Info("Starting backup")
			log.Debug(appliance.GetId())
			apiClient.GetConfig().AddDefaultHeader("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+json", opts.Config.Version))
			run := apiClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, appliance.Id).Authorization(app.Token).InlineObject(iObj)
			res, httpresponse, err := run.Execute()
			if err != nil {
				respBody := backupHTTPResponse{}
				decodeErr := json.NewDecoder(httpresponse.Body).Decode(&respBody)
				if decodeErr != nil {
					return decodeErr
				}
				log.Debug(respBody)
				log.Debug(err)
				return err
			}
			backupID := res.GetId()
			backupIDs[appliance.GetId()] = backupID

			var status string
			backoff := 1 * time.Second
			for status != "done" {
				apiClient.GetConfig().AddDefaultHeader("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+json", opts.Config.Version))
				status, err = getBackupState(ctx, apiClient, app.Token, appliance.Id, backupID)
				if err != nil {
					return err
				}
				// Exponential backoff to not hammer API
				if backoff > opts.Timeout {
					return errors.New("Failed backup. Backup status exceeded timeout.")
				}
				time.Sleep(backoff)
				backoff *= 2
			}

			apiClient.GetConfig().AddDefaultHeader("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
			file, inlineRes, err := apiClient.ApplianceBackupApi.AppliancesIdBackupBackupIdGet(ctx, appliance.Id, backupID).Authorization(app.Token).Execute()
			if err != nil {
				log.Debug(err)
				log.Debug(inlineRes)
				return err
			}
			defer file.Close()
			dst, err := os.Create(fmt.Sprintf("%s/appgate_backup_%s_%s.bkp", opts.Destination, appliance.Name, time.Now().Format("20060102_150405")))
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, file)
			if err != nil {
				return err
			}

			fields = log.Fields{"destination": dst.Name()}
			log.WithFields(fields).Infof("Wrote backup file")

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return backupIDs, err
	}

	return backupIDs, nil
}

func CleanupBackup(opts *BackupOpts, IDs map[string]string) error {
	app, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}

	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	for appID, bckID := range IDs {
		ID := appID
		backupID := bckID
		g.Go(func() error {
			entry := log.WithField("applianceID", ID).WithField("backupID", backupID)
			entry.Info("Cleaning up backup")
			client := app.APIClient
			client.GetConfig().AddDefaultHeader("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
			res, err := app.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdDelete(ctx, ID, backupID).Authorization(opts.Config.GetBearTokenHeaderValue()).Execute()
			if err != nil {
				return err
			}
			entry.Debug(res)
			entry.Info("Done")
			return nil
		})
	}

	return g.Wait()
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

func backupEnabled(ctx context.Context, client *openapi.APIClient, token string) (bool, error) {
	settings, _, err := client.GlobalSettingsApi.GlobalSettingsGet(ctx).Authorization(token).Execute()
	if err != nil {
		return false, err
	}

	return *settings.BackupApiEnabled, nil
}
