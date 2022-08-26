package appliance

import (
	"context"
	"errors"
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
	// WaitForUpgradeStatus does expodential backoff retries on upgrade status until it reaches a desiredStatuses and reports it to current <- string
	WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) error
}

type UpgradeStatus struct {
	Appliance *Appliance
}

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) backoff.Operation {
	name := appliance.GetName()
	logEntry := log.WithField("appliance", name)
	logEntry.WithField("want", desiredStatuses).Info("polling for upgrade status")
	hasRebooted := false
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			if tracker != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					tracker.Update("rebooting, waiting for appliance to come back online")
					hasRebooted = true
				} else if hasRebooted {
					tracker.Update("switching partition")
				} else {
					tracker.Update("installing")
				}
			}
			logEntry.WithError(err).Debug("appliance unreachable")
			return err
		}
		var s string
		details := status.GetDetails()
		if v, ok := status.GetStatusOk(); ok {
			s = *v
			logEntry.WithField("current", s).Debug("received upgrade status")
			if tracker != nil {
				tracker.Update(s)
			}
			if util.InSlice(s, undesiredStatuses) {
				if tracker != nil {
					// send error details for tracker
					tracker.Update(s + " - " + details)
				}
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", name, details))
			}
			if util.InSlice(s, desiredStatuses) {
				logEntry.Info("reached wanted upgrade status")
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
