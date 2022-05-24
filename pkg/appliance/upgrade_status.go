package appliance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v7"
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
	// Wait does expodential backoff retries on upgrade status and return nil if it reaches any of the desiredStatuses
	Wait(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string) error
	// Subscribe does expodential backoff retries on upgrade status until it reaches a desiredStatuses and reports it to current <- string
	Subscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, current chan<- string) error

	Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endState string, failState string, current <-chan string)
}

type UpgradeStatus struct {
	Appliance *Appliance
}

func (u *UpgradeStatus) Wait(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	if err := backoff.Retry(u.upgradeStatus(ctx, appliance, desiredStatuses), b); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string) backoff.Operation {
	fields := log.Fields{"appliance": appliance.GetName()}
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			return err
		}
		var s string
		if v, ok := status.GetStatusOk(); ok {
			s = *v
			responseStatus := status.GetStatus()
			if responseStatus == UpgradeStatusFailed {
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", appliance.GetName(), status.GetDetails()))
			}
		}
		fields["image"] = status.GetDetails()
		fields["status"] = s
		log.WithFields(fields).Infof("waiting for '%s' state", desiredStatuses)
		if util.InSlice(s, desiredStatuses) {
			return nil
		}
		return fmt.Errorf(
			"%s never reached %s, got %q %s",
			appliance.GetName(),
			strings.Join(desiredStatuses, ", "),
			s,
			status.GetDetails(),
		)
	}
}

func (u *UpgradeStatus) upgradeStatusSubscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, currentStatus chan<- string) backoff.Operation {
	fields := log.Fields{"appliance": appliance.GetName()}
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			currentStatus <- "rebooting, waiting for appliance to come back online"
			return err
		}
		var s string
		if v, ok := status.GetStatusOk(); ok {
			s = *v
			responseStatus := status.GetStatus()
			currentStatus <- responseStatus
			if responseStatus == UpgradeStatusFailed {
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", appliance.GetName(), status.GetDetails()))
			}
		}
		fields["image"] = status.GetDetails()
		fields["status"] = s
		log.WithFields(fields).Infof("waiting for '%s' state", desiredStatuses)
		if util.InSlice(s, desiredStatuses) {
			return nil
		}
		return fmt.Errorf(
			"%s never reached %s, got %q %s",
			appliance.GetName(),
			strings.Join(desiredStatuses, ", "),
			s,
			status.GetDetails(),
		)
	}
}

func (u *UpgradeStatus) Subscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, current chan<- string) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return backoff.Retry(u.upgradeStatusSubscribe(ctx, appliance, desiredStatuses, current), b)
	}
}

// Watch starts a new gorotuine with a statusbar for the appliance
func (u *UpgradeStatus) Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endState string, failState string, current <-chan string) {
	go func() {
		log.WithField("appliance", appliance.GetName()).Debug("Watching for appliance state")
		endMsg := "completed"
		previous := ""
		name := appliance.GetName()
		spinner := prompt.AddDefaultSpinner(p, name, "", endMsg)
		barCtxErr := func(bar *mpb.Bar) {
			<-ctx.Done()
			if !bar.Completed() {
				bar.Increment()
			}
		}
		go barCtxErr(spinner)
		for status := range current {
			switch status {
			case endState:
				spinner.Increment()
			case failState:
				if len(previous) > 0 {
					spinner.Abort(false)
				} else if len(status) > 0 && status != previous {
					old := spinner
					spinner = prompt.AddDefaultSpinner(p, name, strings.ReplaceAll(status, "_", " "), endMsg, mpb.BarQueueAfter(old, false))
					go barCtxErr(spinner)
					old.Increment()
					previous = status
				}
			default:
				if len(status) > 0 && status != previous {
					old := spinner
					spinner = prompt.AddDefaultSpinner(p, name, strings.ReplaceAll(status, "_", " "), endMsg, mpb.BarQueueAfter(old, false))
					go barCtxErr(spinner)
					old.Increment()
					previous = status
				}
			}
		}
	}()
}
