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
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	decor "github.com/vbauerster/mpb/v7/decor"
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
	token, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return backupIDs, err
	}
	backupEnabled, err := backupEnabled(ctx, app.APIClient, token, opts.NoInteractive)
	if err != nil {
		if opts.NoInteractive {
			log.WithError(err).Warn("Skipping backup due to error while --no-interactive flag is set")
			return backupIDs, nil
		}
		return backupIDs, fmt.Errorf("Failed to determine backup option: %w", err)
	}
	if !backupEnabled {
		if opts.NoInteractive {
			log.Warn("Skipping backup. Backup API is disabled while --no-interactive flag is set")
			return backupIDs, nil
		}
		return backupIDs, fmt.Errorf("Backup API is disabled in the collective. Use the 'sdpctl appliance backup api' command to enable it.")
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
		if reflect.DeepEqual(opts.FilterFlag, nullFilter) || opts.FilterFlag == nil {
			opts.FilterFlag = util.ParseFilteringFlags(cmd.Flags())
		}

		if opts.PrimaryFlag || opts.NoInteractive {
			pc, err := FindPrimaryController(appliances, hostname)
			if err != nil {
				log.Warn("failed to determine primary controller")
			} else {
				idFilter := []string{}
				if len(opts.FilterFlag["filter"]["id"]) > 0 {
					idFilter = strings.Split(opts.FilterFlag["filter"]["id"], FilterDelimiter)
				}
				idFilter = append(idFilter, pc.GetId())
				opts.FilterFlag["filter"]["id"] = strings.Join(idFilter, FilterDelimiter)
			}
		}

		if opts.CurrentFlag {
			cc, err := FindCurrentController(appliances, hostname)
			if err != nil {
				log.Warn("failed to determine current controller")
			} else {
				idFilter := []string{}
				if len(opts.FilterFlag["filter"]["id"]) > 0 {
					idFilter = strings.Split(opts.FilterFlag["filter"]["id"], FilterDelimiter)
				}
				idFilter = append(idFilter, cc.GetId())
				opts.FilterFlag["filter"]["id"] = strings.Join(idFilter, FilterDelimiter)
			}
		}

		if len(args) > 0 {
			fInclude := []string{}
			if len(opts.FilterFlag["filter"]["name"]) > 0 {
				fInclude = strings.Split(opts.FilterFlag["filter"]["name"], FilterDelimiter)
			}
			fInclude = append(fInclude, args...)
			opts.FilterFlag["filter"]["name"] = strings.Join(fInclude, FilterDelimiter)
		}

		if !reflect.DeepEqual(nullFilter, opts.FilterFlag) {
			toBackup = append(toBackup, FilterAppliances(appliances, opts.FilterFlag)...)
		}
	}

	if len(toBackup) <= 0 {
		toBackup, err = BackupPrompt(appliances)
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
			spinner := util.AddDefaultSpinner(p, appliance.GetName(), "preparing backup", "")
			run := apiClient.ApplianceBackupApi.AppliancesIdBackupPost(ctx, appliance.Id).Authorization(app.Token).InlineObject(iObj)
			res, httpresponse, err := run.Execute()
			if err != nil {
				spinner.Abort(true)
				respBody := backupHTTPResponse{}
				decodeErr := json.NewDecoder(httpresponse.Body).Decode(&respBody)
				if decodeErr != nil {
					return decodeErr
				}
				log.WithError(err).Error("Caught backup error")
				return err
			}
			backupID := res.GetId()
			bIDChan <- map[string]string{appliance.GetId(): backupID}

			var status string
			backoff := 1 * time.Second
			for status != "done" {
				time.Sleep(backoff)
				backoff *= 2
				currentStatus, err := getBackupState(ctx, apiClient, app.Token, appliance.Id, backupID)
				if err != nil {
					spinner.Abort(true)
					return err
				}
				if currentStatus != status {
					old := spinner
					old.Increment()
					spinner = util.AddDefaultSpinner(p, appliance.GetName(), currentStatus, "", mpb.BarQueueAfter(old, false))
				}
				status = currentStatus
				// Exponential backoff to not hammer API
				if backoff > opts.Timeout {
					spinner.Abort(true)
					return errors.New("Failed backup. Backup status exceeded timeout.")
				}
			}
			old := spinner
			old.Increment()
			spinner = util.AddDefaultSpinner(p, appliance.GetName(), "preparing download", "", mpb.BarQueueAfter(old, false))
			ctxWithGPGAccept := context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
			file, inlineRes, err := apiClient.ApplianceBackupApi.AppliancesIdBackupBackupIdGet(ctxWithGPGAccept, appliance.Id, backupID).Authorization(app.Token).Execute()
			if err != nil {
				log.WithError(err).WithField("response", inlineRes).Debug(err)
				return err
			}
			defer file.Close()
			fileStat, err := file.Stat()
			if err != nil {
				return err
			}
			dst, err := os.Create(fmt.Sprintf("%s/appgate_backup_%s_%s.bkp", opts.Destination, appliance.Name, time.Now().Format("20060102_150405")))
			if err != nil {
				return err
			}
			defer dst.Close()

			spinner.Increment()
			limitReader := io.LimitReader(file, fileStat.Size())
			name := filepath.Base(dst.Name())
			bar := p.AddBar(fileStat.Size(), mpb.BarQueueAfter(spinner, false), mpb.BarWidth(50),
				mpb.BarFillerOnComplete("downloaded"),
				mpb.PrependDecorators(
					decor.OnComplete(decor.Name(" downloading "), " ✓ "),
					decor.Name(name, decor.WCSyncWidthR),
				),
				mpb.AppendDecorators(
					decor.OnComplete(decor.CountersKibiByte("% .2f / % .2f"), ""),
					decor.OnComplete(decor.Name(" | "), ""),
					decor.OnComplete(decor.AverageSpeed(decor.UnitKiB, "% .2f"), ""),
				),
			)
			proxyReader := bar.ProxyReader(limitReader)
			defer proxyReader.Close()

			_, err = io.Copy(dst, proxyReader)
			if err != nil {
				return err
			}

			log.WithField("file", dst.Name()).Info("Wrote backup file")

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

func BackupPrompt(appliances []openapi.Appliance) ([]openapi.Appliance, error) {
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
	if err := prompt.SurveyAskOne(qs, &selected); err != nil {
		return nil, err
	}
	log.WithField("appliances", selected)

	result := FilterAppliances(appliances, map[string]map[string]string{
		"filter": {
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
			Message: "Do you want to enable it now?",
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
