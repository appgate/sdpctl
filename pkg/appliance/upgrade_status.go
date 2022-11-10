package appliance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
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

type UpgradeStatusCtx string

const UpgradeStatusGetErrorMessage UpgradeStatusCtx = "UpgradeStatusGetErrorMessage"

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) backoff.Operation {
	name := appliance.GetName()
	logEntry := log.WithField("appliance", name)
	logEntry.WithField("want", desiredStatuses).Info("Polling for upgrade status")
	return func() error {
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			if tracker != nil {
				msg := "no response, appliance offline"
				if v, ok := ctx.Value(UpgradeStatusGetErrorMessage).(string); ok {
					msg = v
				}
				tracker.Update(msg)
			}
			logEntry.WithError(err).Debug("Appliance unreachable")
			return err
		}
		var s string
		details := status.GetDetails()
		if v, ok := status.GetStatusOk(); ok {
			s = *v
			logEntry.WithField("current", s).Debug("Received upgrade status")
			if tracker != nil {
				tracker.Update(s)
			}
			if util.InSlice(s, undesiredStatuses) {
				if tracker != nil {
					// send error details for tracker
					tracker.Fail(s + " - " + details)
				}
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", name, details))
			}
			if util.InSlice(s, desiredStatuses) {
				logEntry.Info("Reached wanted upgrade status")
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
