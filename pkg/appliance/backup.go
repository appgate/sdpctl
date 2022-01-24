package appliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/api"
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
	Config        *configuration.Config
	Appliance     func(*configuration.Config) (*Appliance, error)
	Out           io.Writer
	Destination   string
	NotifyURL     string
	Include       []string
	AllFlag       bool
	PrimaryFlag   bool
	CurrentFlag   bool
	NoInteraction bool
	Timeout       time.Duration
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

func PerformBackup(cmd *cobra.Command, args []string, opts *BackupOpts) (map[string]string, error) {
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

	backupEnabled, err := backupEnabled(ctx, app.APIClient, opts.Config.GetBearTokenHeaderValue(), opts.NoInteraction)
	if err != nil {
		return backupIDs, fmt.Errorf("Failed to determine backup option: %w", err)
	}
	if !backupEnabled {
		return backupIDs, fmt.Errorf("Backup API is disabled in the collective. Use the 'appgatectl appliance backup api' command to enable it.")
	}

	appliances, err := app.List(ctx, nil)
	if err != nil {
		return backupIDs, err
	}

	var toBackup []openapi.Appliance
	if opts.AllFlag {
		toBackup = appliances
	} else {
		hostname, _ := opts.Config.GetHost()
		nullFilter := map[string]map[string]string{
			"filter":  {},
			"exclude": {},
		}
		filter := util.ParseFilteringFlags(cmd.Flags())

		if opts.PrimaryFlag {
			pc, err := FindPrimaryController(appliances, hostname)
			if err != nil {
				log.Warn("failed to determine primary controller")
			} else {
				idFilter := []string{}
				if len(filter["filter"]["id"]) > 0 {
					idFilter = strings.Split(filter["filter"]["id"], FilterDelimiter)
				}
				idFilter = append(idFilter, pc.GetId())
				filter["filter"]["id"] = strings.Join(idFilter, FilterDelimiter)
			}
		}

		if opts.CurrentFlag {
			cc, err := FindCurrentController(appliances, hostname)
			if err != nil {
				log.Warn("failed to determine current controller")
			} else {
				idFilter := []string{}
				if len(filter["filter"]["id"]) > 0 {
					idFilter = strings.Split(filter["filter"]["id"], FilterDelimiter)
				}
				idFilter = append(idFilter, cc.GetId())
				filter["filter"]["id"] = strings.Join(idFilter, FilterDelimiter)
			}
		}

		if len(args) > 0 {
			fInclude := []string{}
			if len(filter["filter"]["name"]) > 0 {
				fInclude = strings.Split(filter["filter"]["name"], FilterDelimiter)
			}
			fInclude = append(fInclude, args...)
			filter["filter"]["name"] = strings.Join(fInclude, FilterDelimiter)
		}

		if !reflect.DeepEqual(nullFilter, filter) {
			toBackup = append(toBackup, FilterAppliances(appliances, filter)...)
		}
	}

	if len(toBackup) <= 0 {
		toBackup = backupPrompt(appliances)
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
			fields := log.Fields{"appliance": appliance.Name, "id": appliance.Id}
			log.WithFields(fields).Info("Starting backup")
			run := apiClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, appliance.Id).Authorization(app.Token).InlineObject(iObj)
			res, httpresponse, err := run.Execute()
			if err != nil {
				respBody := backupHTTPResponse{}
				decodeErr := json.NewDecoder(httpresponse.Body).Decode(&respBody)
				if decodeErr != nil {
					return decodeErr
				}
				log.Debug(err)
				return err
			}
			backupID := res.GetId()
			backupIDs[appliance.GetId()] = backupID

			var status string
			backoff := 1 * time.Second
			for status != "done" {
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

			ctxWithGPGAccept := context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
			file, inlineRes, err := apiClient.ApplianceBackupApi.AppliancesIdBackupBackupIdGet(ctxWithGPGAccept, appliance.Id, backupID).Authorization(app.Token).Execute()
			if err != nil {
				log.WithField("error", err).WithField("response", inlineRes).Debug(err)
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

	ctxWithGPGAccept := context.WithValue(context.Background(), openapi.ContextAcceptHeader, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
	g, ctx := errgroup.WithContext(ctxWithGPGAccept)
	for appID, bckID := range IDs {
		ID := appID
		backupID := bckID
		g.Go(func() error {
			entry := log.WithField("applianceID", ID).WithField("backupID", backupID)
			entry.Info("Cleaning up backup")
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

func backupPrompt(appliances []openapi.Appliance) []openapi.Appliance {
	names := []string{}

	for _, a := range appliances {
		names = append(names, a.GetName())
	}

	qs := &survey.MultiSelect{
		PageSize: len(appliances),
		Message:  "select appliances to backup:",
		Options:  names,
	}
	var selected []string
	survey.AskOne(qs, &selected)
	log.WithField("appliances", selected)

	result := FilterAppliances(appliances, map[string]map[string]string{
		"filter": {
			"name": strings.Join(selected, FilterDelimiter),
		},
	})

	return result
}

func getBackupState(ctx context.Context, client *openapi.APIClient, token string, aID string, bID string) (string, error) {
	res, _, err := client.ApplianceBackupApi.AppliancesIdBackupBackupIdStatusGet(ctx, aID, bID).Authorization(token).Execute()
	if err != nil {
		log.Debug(err)
		return "", err
	}
	log.WithField("appliance", aID).WithField("current state", res.GetStatus()).Debug("Waiting for backup to reach done state")

	return *res.Status, nil
}

func backupEnabled(ctx context.Context, client *openapi.APIClient, token string, noInteractionFlag bool) (bool, error) {
	enable := true
	settings, _, err := client.GlobalSettingsApi.GlobalSettingsGet(ctx).Authorization(token).Execute()
	if err != nil {
		return false, err
	}

	if !*settings.BackupApiEnabled {
		log.Warn("Backup API is disabled.")
		if !noInteractionFlag {
			q := &survey.Confirm{
				Message: "Do you want to enable it now?",
				Default: enable,
			}
			if err := survey.AskOne(q, &enable, survey.WithValidator(survey.Required)); err != nil {
				return false, err
			}
		}
	}

	if enable && !noInteractionFlag {
		var password string
		p := &survey.Password{
			Message: "Enter passphrase for backups: ",
		}
		if err := survey.AskOne(p, &password, survey.WithValidator(survey.Required)); err != nil {
			return false, err
		}
		settings.SetBackupApiEnabled(true)
		settings.SetBackupPassphrase(password)
		response, err := client.GlobalSettingsApi.GlobalSettingsPut(ctx).GlobalSettings(settings).Authorization(token).Execute()
		if err != nil {
			return false, api.HTTPErrorResponse(response, err)
		}
		log.Info("Backup API enabled")
	}

	return enable, nil
}
