package appliance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/appliance/backup"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	DefaultBackupDestination = filepath.Join(filesystem.DownloadDir(), "backup")
)

type BackupOpts struct {
	Config        *configuration.Config
	Appliance     func(*configuration.Config) (*Appliance, error)
	Out           io.Writer
	SpinnerOut    func() io.Writer
	Destination   string
	With          []string
	AllFlag       bool
	PrimaryFlag   bool
	CurrentFlag   bool
	NoInteractive bool
	FilterFlag    map[string]map[string]string
	OrderBy       []string
	Descending    bool
	Quiet         bool
	CiMode        bool
}

func PrepareBackup(opts *BackupOpts) error {
	log.WithField("destination", opts.Destination).Info("Preparing backup")

	if IsOnAppliance() {
		return fmt.Errorf("This should not be executed on an appliance")
	}

	opts.Destination = filesystem.AbsolutePath(opts.Destination)
	if err := os.MkdirAll(opts.Destination, 0700); err != nil {
		return err
	}

	return nil
}

func PerformBackup(cmd *cobra.Command, args []string, opts *BackupOpts) (map[string]string, error) {
	spinnerOut := opts.SpinnerOut()
	backupIDs := make(map[string]string)
	ctx := context.Background()

	var err error
	opts.CiMode, err = cmd.Flags().GetBool("ci-mode")
	if err != nil {
		return nil, err
	}

	audit := util.InSlice("audit", opts.With)
	logs := util.InSlice("logs", opts.With)

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
		// This most likely means that the user does not have permission to read global settings, which is fine, so we skip the entire enable/disable prompt
		if errors.Is(err, api.ForbiddenErr) {
			goto NO_ENABLE_CHECK
		}
		if opts.NoInteractive {
			return backupIDs, errors.New("Backup failed due to error while --no-interactive flag is set")
		}
		return backupIDs, fmt.Errorf("Failed to determine backup option: %w", err)
	}
	if !backupEnabled {
		if opts.NoInteractive {
			return backupIDs, errors.New("Using '--no-interactive' flag while Backup API is disabled. Use the 'sdpctl appliance backup api' command to enable it before trying again")
		}
		return backupIDs, errors.New("Backup API is disabled in the collective. Use the 'sdpctl appliance backup api' command to enable it")
	}

NO_ENABLE_CHECK:
	appliances, err := app.List(ctx, nil, []string{"name"}, false)
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
			opts.FilterFlag, opts.OrderBy, opts.Descending = util.ParseFilteringFlags(cmd.Flags(), DefaultCommandFilter)
		}

		if opts.PrimaryFlag {
			pc, err := FindPrimaryController(appliances, hostname, false)
			if err != nil {
				log.Warn("Failed to determine the primary Controller")
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
				log.Warn("Failed to determine the current Controller")
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
			res, _, err := FilterAppliances(appliances, opts.FilterFlag, opts.OrderBy, opts.Descending)
			if err != nil {
				return nil, err
			}
			toBackup = append(toBackup, res...)
		}
	}

	if len(toBackup) <= 0 {
		if opts.NoInteractive {
			return nil, errors.New("No appliances to backup and prompt not available due to '--no-interactive' flag is set")
		}
		toBackup, err = BackupPrompt(appliances, []openapi.Appliance{})
		if err != nil {
			return nil, err
		}
	}

	// Filter offline appliances
	initialStats, _, err := app.Stats(ctx, nil, opts.OrderBy, opts.Descending)
	if err != nil {
		return backupIDs, err
	}
	toBackup, offline, _ := FilterAvailable(toBackup, initialStats.GetData())

	for _, v := range offline {
		log.WithField("appliance", v.GetName()).Info("Skipping appliance. Appliance is offline")
	}

	if len(toBackup) <= 0 {
		fmt.Fprintln(opts.Out, "No appliances to backup. Either no appliance was selected or the selected appliances are offline")
		return nil, nil
	}

	if !opts.Quiet {
		msg, err := showBackupSummary(opts.Destination, toBackup)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(opts.Out, "%s\n", msg)
	}

	type backedUp struct {
		applianceID, backupID, destination string
	}

	var (
		wg           sync.WaitGroup
		count        = len(toBackup)
		backups      = make(chan backedUp, count)
		errorChannel = make(chan error, count)
		backupAPI    = backup.New(app.HTTPClient, app.APIClient, opts.Config, app.Token)
		progressBars *tui.Progress
	)

	progressBars = tui.New(ctx, spinnerOut)
	defer progressBars.Wait()

	wg.Add(count)

	retryStatus := func(ctx context.Context, applianceID, backupID string, tracker *tui.Tracker) error {
		bo := backoff.NewExponentialBackOff()
		bo.MaxElapsedTime = 0
		networkErrors := 0
		return backoff.Retry(func() error {
			status, err := backupAPI.Status(ctx, applianceID, backupID)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					networkErrors++
					if networkErrors >= 5 {
						return backoff.Permanent(cmdutil.ErrNetworkError)
					}
				}
				return err
			}
			networkErrors = 0
			message := status.GetStatus()
			if len(status.GetMessage()) > 0 {
				message = status.GetMessage()
			}
			log.WithFields(log.Fields{
				"appliance_id": applianceID,
				"backup_id":    backupID,
				"status":       status.GetStatus(),
			}).Info(message)
			tracker.Update(message)
			if status.GetStatus() != backup.Done {
				return fmt.Errorf("Backup not done for appliance %s, got %s", applianceID, status.GetStatus())
			}
			if _, ok := status.GetResultOk(); ok && status.GetResult() != backup.Success {
				return backoff.Permanent(fmt.Errorf("%s %s", status.GetResult(), status.GetOutput()))
			}
			return nil
		}, bo)
	}

	b := func(appliance openapi.Appliance, tracker *tui.Tracker) (backedUp, error) {
		b := backedUp{applianceID: appliance.GetId()}
		f := log.Fields{
			"appliance_id": b.applianceID,
		}
		logger := log.WithFields(f)
		b.backupID, err = backupAPI.Initiate(ctx, b.applianceID, logs, audit)
		if err != nil {
			if errors.Is(err, api.UnavailableErr) {
				return b, errors.New("The backup API is disabled")
			}
			return b, err
		}
		logger = logger.WithField("backup_id", b.backupID)
		logger.Info("Initiated backup")
		tracker.Update("Initiated backup")
		if err := retryStatus(ctx, b.applianceID, b.backupID, tracker); err != nil {
			logger.WithError(err).Error("backup failed")
			return b, err
		}
		msg := "downloading"
		logger.Info(msg)
		tracker.Update(msg)
		b.destination = filepath.Join(opts.Destination, fmt.Sprintf("appgate_backup_%s_%s.bkp", strings.ReplaceAll(appliance.GetName(), " ", "_"), time.Now().Format("20060102_150405")))
		file, err := backupAPI.ChunkedDownload(ctx, b.applianceID, b.backupID, b.destination)
		if err != nil {
			logger.WithError(err).Error("backup failed")
			return b, err
		}
		logger = logger.WithField("download_path", file.Name())
		logger.Info("download complete")
		tracker.Update("download complete")
		return b, nil
	}

	for _, a := range toBackup {
		t := progressBars.AddTracker(a.GetName(), "waiting", "download complete")
		go t.Watch([]string{"download complete"}, []string{backup.Failure})
		go func(appliance openapi.Appliance, tracker *tui.Tracker) {
			defer wg.Done()
			backedUp, err := b(appliance, t)
			if err != nil {
				errorChannel <- fmt.Errorf("Backup failed for %s: %s", appliance.GetName(), err)
				return
			}
			backups <- backedUp
		}(a, t)
	}

	go func() {
		wg.Wait()
		close(backups)
		close(errorChannel)
	}()

	for b := range backups {
		backupIDs[b.applianceID] = b.backupID
		log.WithFields(log.Fields{
			"file":         b.destination,
			"appliance_id": b.applianceID,
			"backup_id":    b.backupID,
		}).Info("Wrote backup file")
	}
	var result *multierror.Error
	for err := range errorChannel {
		result = multierror.Append(err)
	}

	return backupIDs, result.ErrorOrNil()
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
	for appID, bckID := range IDs {
		ID := appID
		backupID := bckID
		logger := log.WithFields(log.Fields{"appliance_id": appID, "backup_id": bckID})
		logger.Info("cleaning up")
		g.Go(func() error {
			res, err := app.APIClient.ApplianceBackupApi.AppliancesIdBackupBackupIdDelete(ctx, ID, backupID).Authorization(token).Execute()
			if err != nil {
				logger.WithError(err).Error("cleanup failed")
				return api.HTTPErrorResponse(res, err)
			}
			logger.Info("cleanup complete")
			return nil
		})
	}
	g.Wait()
	fmt.Fprint(opts.Out, "Backup complete!\n\n")

	return nil
}

func BackupPrompt(appliances []openapi.Appliance, preSelected []openapi.Appliance) ([]openapi.Appliance, error) {
	names := []string{}
	preSelectNames := []string{}

	selectorNameMap := map[string]string{}
	appendFunctions := func(appliance openapi.Appliance) string {
		name := appliance.GetName()
		activeFunctions := GetActiveFunctions(appliance)
		selectorName := fmt.Sprintf("%s ( %s )", name, strings.Join(activeFunctions, ", "))
		selectorNameMap[selectorName] = name
		return selectorName
	}

	// Filter out all but Controllers, LogServers and Portals
	appliances, _, err := FilterAppliances(appliances, map[string]map[string]string{
		"include": {"function": strings.Join([]string{FunctionController, FunctionLogServer, FunctionPortal}, FilterDelimiter)},
	}, []string{"name"}, false)
	if err != nil {
		return nil, err
	}

	for _, a := range appliances {
		selectorName := appendFunctions(a)
		for _, ps := range preSelected {
			if a.GetName() == ps.GetName() {
				preSelectNames = append(preSelectNames, selectorName)
			}
		}
		names = append(names, selectorName)
	}

	qs := &survey.MultiSelect{
		PageSize: len(appliances),
		Message:  "select appliances to backup:",
		Options:  names,
		Default:  preSelectNames,
	}
	var selectedEntries []string
	if err := prompt.SurveyAskOne(qs, &selectedEntries); err != nil {
		return nil, err
	}
	selected := []string{}
	for _, selectorName := range selectedEntries {
		selected = append(selected, selectorNameMap[selectorName])
	}
	log.WithField("appliances", selected).Info("selected appliances for backup")

	result, _, err := FilterAppliances(appliances, map[string]map[string]string{
		"include": {
			"name": strings.Join(selected, FilterDelimiter),
		},
	}, []string{"name"}, false)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func backupEnabled(ctx context.Context, client *openapi.APIClient, token string, noInteraction bool) (bool, error) {
	settings, response, err := client.GlobalSettingsApi.GlobalSettingsGet(ctx).Authorization(token).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusForbidden {
			return false, api.ForbiddenErr
		}
		return false, api.HTTPErrorResponse(response, err)
	}
	enabled := settings.GetBackupApiEnabled()
	if !enabled && !noInteraction {
		log.Warn("Backup API is disabled on the appliance")
		var shouldEnable bool
		q := &survey.Confirm{
			Message: "Backup API is disabled on the appliance. Do you want to enable it now?",
			Default: true,
		}
		if err := prompt.SurveyAskOne(q, &shouldEnable, survey.WithValidator(survey.Required)); err != nil {
			return false, err
		}

		if shouldEnable {
			settings.SetBackupApiEnabled(true)
			password, err := prompt.PasswordConfirmation("The passphrase to encrypt the appliance backups when the Backup API is used:")
			if err != nil {
				return false, err
			}
			settings.SetBackupPassphrase(password)
			result, err := client.GlobalSettingsApi.GlobalSettingsPut(ctx).GlobalSettings(*settings).Authorization(token).Execute()
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
