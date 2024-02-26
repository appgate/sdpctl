package appliance

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
)

const backoffDefaultTimeout = time.Minute * 30

var defaultExponentialBackOff = &backoff.ExponentialBackOff{
	InitialInterval:     1 * time.Second,
	Multiplier:          2,
	RandomizationFactor: 0.7,
	MaxInterval:         10 * time.Second,
	MaxElapsedTime:      backoffDefaultTimeout,
	Stop:                backoff.Stop,
	Clock:               backoff.SystemClock,
}

type WaitForUpgradeStatus interface {
	// WaitForUpgradeStatus does exponential backoff retries on upgrade status until it reaches a desiredStatuses and reports it to current <- string
	WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) error
}

type UpgradeStatus struct {
	Appliance *Appliance
}

type IsPrimaryUpgrade string
type CalledAs string

const (
	PrimaryUpgrade IsPrimaryUpgrade = "IsPrimaryUpgrade"
	Caller         CalledAs         = "calledAs"
)

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) backoff.Operation {
	name := appliance.GetName()
	logEntry := log.WithField("appliance", name)
	logEntry.WithField("want", desiredStatuses).Info("Polling for upgrade status")
	hasRebooted := false
	offlineRegex := regexp.MustCompile(`No response Get`)
	onlineRegex := regexp.MustCompile(`Bad Gateway`)
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			if tracker != nil {
				msg := tracker.Current()

				// Check if upgrading primary controller and apply logic for offline check
				if primaryUpgrade, ok := ctx.Value(PrimaryUpgrade).(bool); ok && primaryUpgrade {
					msg = "switching partition"
					if offlineRegex.MatchString(err.Error()) {
						hasRebooted = true
					} else if onlineRegex.MatchString(err.Error()) {
						if hasRebooted {
							msg = "initializing"
						} else {
							msg = "installing"
						}
					} else {
						msg = err.Error()
					}
				} else if !errors.Is(err, context.DeadlineExceeded) {
					msg = err.Error()
				}

				tracker.Update(msg)
			}
			logEntry.WithError(err).Debug("No response, appliance offline")
			return err
		}
		var s string
		details := status.GetDetails()
		if v, ok := status.GetStatusOk(); ok {
			s = *v
			logEntry.WithField("current", s).Debug("Received status")
			if tracker != nil {
				tracker.Update(s)
			}
			if util.InSlice(s, undesiredStatuses) {
				if tracker != nil {
					// send error details for tracker
					tracker.Fail(s + " - " + details)
				}
				err := fmt.Errorf("Command failed on %s %s %s", name, s, details)
				if calledAs, ok := ctx.Value(Caller).(string); ok {
					err = fmt.Errorf("%s failed on %s %s %s", calledAs, name, s, details)
				}
				logEntry.WithError(err).WithFields(log.Fields{"status": s, "details": details}).Error("Unwanted status on appliance")
				return backoff.Permanent(err)
			}
			if util.InSlice(s, desiredStatuses) {
				logEntry.Info("Reached wanted status")
				return nil
			}
		}
		return fmt.Errorf(
			"%s never reached %s, got %q %s",
			name,
			strings.Join(desiredStatuses, ", "),
			s,
			details,
		)
	}
}

func (u *UpgradeStatus) WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return backoff.Retry(u.upgradeStatus(ctx, appliance, desiredStatuses, undesiredStatuses, tracker), b)
	}
}
