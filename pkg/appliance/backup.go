package appliance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	mpb "github.com/vbauerster/mpb/v7"
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
	With          []string
	AllFlag       bool
	PrimaryFlag   bool
	CurrentFlag   bool
	Timeout       time.Duration
	NoInteractive bool
	FilterFlag    map[string]map[string]string
	Quiet         bool
}

type backupHTTPResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func PrepareBackup(opts *BackupOpts) error {
	log.WithField("destination", opts.Destination).Info("Preparing backup")

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
	aud := util.InSlice("audit", opts.With)
	logs := util.InSlice("logs", opts.With)

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
	token, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return backupIDs, err
	}
	backupEnabled, err := backupEnabled(ctx, app.APIClient, token, opts.NoInteractive)
	if err != nil {
		if opts.NoInteractive {
			return backupIDs, errors.New("Backup failed due to error while --no-interactive flag is set")
		}
		return backupIDs, fmt.Errorf("Failed to determine backup option: %w", err)
	}
	if !backupEnabled {
		if opts.NoInteractive {
			return backupIDs, errors.New("Using '--no-interactive' flag while backup API is disabled. Use the 'sdpctl appliance backup api' command to enable it before trying again.")
		}
		return backupIDs, errors.New("Backup API is disabled in the collective. Use the 'sdpctl appliance backup api' command to enable it.")
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
			"include": {},
			"exclude": {},
		}
		if reflect.DeepEqual(opts.FilterFlag, nullFilter) || opts.FilterFlag == nil {
			opts.FilterFlag = util.ParseFilteringFlags(cmd.Flags(), DefaultCommandFilter)
		}

		if opts.PrimaryFlag || opts.NoInteractive {
			pc, err := FindPrimaryController(appliances, hostname)
			if err != nil {
				log.Warn("failed to determine primary controller")
			} else {
				idFilter := []string{}
				if len(opts.FilterFlag["include"]["id"]) > 0 {
					idFilter = strings.Split(opts.FilterFlag["include"]["id"], FilterDelimiter)
				}
				idFilter = append(idFilter, pc.GetId())
				opts.FilterFlag["include"]["id"] = strings.Join(idFilter, FilterDelimiter)
			}
		}

		if opts.CurrentFlag {
			cc, err := FindCurrentController(appliances, hostname)
			if err != nil {
				log.Warn("failed to determine current controller")
			} else {
				idFilter := []string{}
				if len(opts.FilterFlag["include"]["id"]) > 0 {
					idFilter = strings.Split(opts.FilterFlag["include"]["id"], FilterDelimiter)
				}
				idFilter = append(idFilter, cc.GetId())
				opts.FilterFlag["include"]["id"] = strings.Join(idFilter, FilterDelimiter)
			}
		}

		if len(args) > 0 {
			fInclude := []string{}
			if len(opts.FilterFlag["include"]["name"]) > 0 {
				fInclude = strings.Split(opts.FilterFlag["include"]["name"], FilterDelimiter)
			}
			fInclude = append(fInclude, args...)
			opts.FilterFlag["include"]["name"] = strings.Join(fInclude, FilterDelimiter)
		}

		if !reflect.DeepEqual(nullFilter, opts.FilterFlag) {
			toBackup = append(toBackup, FilterAppliances(appliances, opts.FilterFlag)...)
		}
	}

	if len(toBackup) <= 0 {
		toBackup, err = BackupPrompt(appliances, []openapi.Appliance{})
		if err != nil {
			return nil, err
		}
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

	if len(toBackup) <= 0 {
		fmt.Fprintln(opts.Out, "No appliances to backup. Either no appliance was selected or the selected appliances are offline.")
		return nil, nil
	}

	if !opts.Quiet {
		msg, err := showBackupSummary(opts.Destination, toBackup)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(opts.Out, "%s\n", msg)
	}
	g, ctx := errgroup.WithContext(ctx)
	backupCount := len(toBackup)
	p := mpb.NewWithContext(ctx)
	log.Infof("Starting backup on %d appliances", backupCount)
	bIDChan := make(chan map[string]string, backupCount)
	for _, a := range toBackup {
		appliance := a
		apiClient := app.APIClient
		g.Go(func() error {
			spinner := util.AddDefaultSpinner(p, appliance.GetName(), "backing up", "completed")
			log.WithField("appliance", appliance.GetName()).Info("backing up")
			run := apiClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, appliance.Id).Authorization(app.Token).InlineObject(iObj)
			res, httpresponse, err := run.Execute()
			if err != nil {
				spinner.Abort(false)
				respBody := backupHTTPResponse{}
				decodeErr := json.NewDecoder(httpresponse.Body).Decode(&respBody)
				if decodeErr != nil {
					return decodeErr
				}
				log.WithError(err).Error("Caught backup error")
				return err
			}
			backupID := res.GetId()
			log.WithField("backup_id", backupID).Info("recieved backup id")
			bIDChan <- map[string]string{appliance.GetId(): backupID}

			var status string
			var backoff float32 = 1
			now := time.Now()
			log.WithField("appliance", appliance.GetName()).Info("waiting for backup to be ready")
			for status != "done" {
				time.Sleep(time.Duration(backoff) * time.Second)
				if backoff < 5 {
					backoff *= 1.2
				}
				log.WithField("backoff", backoff).Debug("backoff request")
				currentStatus, err := getBackupState(ctx, apiClient, app.Token, appliance.Id, backupID)
				if err != nil {
					return err
				}
				status = currentStatus
				// Exponential backoff to not hammer API
				if time.Since(now) > opts.Timeout {
					return errors.New("Failed backup. Backup status exceeded timeout.")
				}
			}
			log.WithField("backup_id", backupID).Info("recieved backup")
			ctxWithGPGAccept := context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
			file, inlineRes, err := apiClient.ApplianceBackupApi.AppliancesIdBackupBackupIdGet(ctxWithGPGAccept, appliance.Id, backupID).Authorization(app.Token).Execute()
			if err != nil {
				spinner.Abort(false)
				log.WithError(err).WithField("response", inlineRes).Debug(err)
				return err
			}
			defer file.Close()
			dst := fmt.Sprintf("%s/appgate_backup_%s_%s.bkp", opts.Destination, appliance.Name, time.Now().Format("20060102_150405"))

			err = os.Rename(file.Name(), dst)
			if err != nil {
				spinner.Abort(false)
				return err
			}

			log.WithField("file", dst).Info("Wrote backup file")
			spinner.Increment()
			spinner.Wait()
			return nil
		})
	}

	go func() {
		g.Wait()
		close(bIDChan)
	}()

	if err := g.Wait(); err != nil {
		return backupIDs, err
	}

	for bID := range bIDChan {
		for key, id := range bID {
			backupIDs[key] = id
		}
	}
	p.Wait()
	return backupIDs, nil
}

func CleanupBackup(opts *BackupOpts, IDs map[string]string) error {
	if IDs == nil || len(IDs) <= 0 {
		return errors.New("Command finished, but no appliances were backed up. See log for more details")
	}
	app, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}
	token, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	ctxWithGPGAccept := context.WithValue(context.Background(), openapi.ContextAcceptHeader, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
	g, ctx := errgroup.WithContext(ctxWithGPGAccept)
	log.WithField("backup_ids", IDs).Info("Cleaning up...")
	for appID, bckID := range IDs {
		ID := appID
		backupID := bckID
		g.Go(func() error {
			entry := log.WithField("applianceID", ID).WithField("backupID", backupID)
			res, err := app.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdDelete(ctx, ID, backupID).Authorization(token).Execute()
			if err != nil {
				return err
			}
			entry.Debug(res)
			return nil
		})
	}
	log.Info("Finished cleanup")
	fmt.Fprint(opts.Out, "Backup complete!\n\n")

	return g.Wait()
}

func BackupPrompt(appliances []openapi.Appliance, preSelected []openapi.Appliance) ([]openapi.Appliance, error) {
	names := []string{}
	preSelectNames := []string{}

	for _, a := range appliances {
		names = append(names, a.GetName())
	}
	for _, a := range preSelected {
		preSelectNames = append(preSelectNames, a.GetName())
	}

	qs := &survey.MultiSelect{
		PageSize: len(appliances),
		Message:  "select appliances to backup:",
		Options:  names,
		Default:  preSelectNames,
	}
	var selected []string
	if err := prompt.SurveyAskOne(qs, &selected); err != nil {
		return nil, err
	}
	log.WithField("appliances", selected)

	result := FilterAppliances(appliances, map[string]map[string]string{
		"include": {
			"name": strings.Join(selected, FilterDelimiter),
		},
	})

	return result, nil
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

func backupEnabled(ctx context.Context, client *openapi.APIClient, token string, noInteraction bool) (bool, error) {
	settings, _, err := client.GlobalSettingsApi.GlobalSettingsGet(ctx).Authorization(token).Execute()
	if err != nil {
		return false, err
	}
	enabled := settings.GetBackupApiEnabled()
	if !enabled && !noInteraction {
		log.Warn("Backup API is disabled on the appliance.")
		var shouldEnable bool
		q := &survey.Confirm{
			Message: "Backup API is disabled on the appliance. Do you want to enable it now?",
			Default: true,
		}
		if err := survey.AskOne(q, &shouldEnable, survey.WithValidator(survey.Required)); err != nil {
			return false, err
		}

		if shouldEnable {
			settings.SetBackupApiEnabled(true)
			password, err := prompt.PasswordConfirmation("The passphrase to encrypt Appliance Backups when backup API is used:")
			if err != nil {
				return false, err
			}
			settings.SetBackupPassphrase(password)
			result, err := client.GlobalSettingsApi.GlobalSettingsPut(ctx).GlobalSettings(settings).Authorization(token).Execute()
			if err != nil {
				return false, api.HTTPErrorResponse(result, err)
			}
			newSettings, response, err := client.GlobalSettingsApi.GlobalSettingsGet(ctx).Authorization(token).Execute()
			if err != nil {
				return false, api.HTTPErrorResponse(response, err)
			}
			enabled = newSettings.GetBackupApiEnabled()
		}
	}

	return enabled, nil
}

func showBackupSummary(dest string, appliances []openapi.Appliance) (string, error) {
	type ApplianceStub struct {
		Name string
		ID   string
	}
	type SummaryStub struct {
		Appliances  []ApplianceStub
		Destination string
	}

	const message = `
Will perform backup on the following appliances:

{{- range .Appliances }}
 - {{ .Name -}}
{{ end }}

Backup destination is {{ .Destination }}
`

	data := SummaryStub{Destination: dest}
	for _, app := range appliances {
		data.Appliances = append(data.Appliances, ApplianceStub{
			Name: app.GetName(),
			ID:   app.GetId(),
		})
	}

	t := template.Must(template.New("").Parse(message))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
