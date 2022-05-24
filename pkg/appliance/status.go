package appliance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v7"
)

const DefaultTimeout = time.Minute * 30

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

var defaultExponentialBackOff = &backoff.ExponentialBackOff{
	InitialInterval:     1 * time.Second,
	Multiplier:          2,
	RandomizationFactor: 0.7,
	MaxInterval:         10 * time.Second,
	MaxElapsedTime:      DefaultTimeout,
	Stop:                backoff.Stop,
	Clock:               backoff.SystemClock,
}

func (u *UpgradeStatus) upgradeStatusSubscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, currentStatus chan<- string) backoff.Operation {
	name := appliance.GetName()
	fields := log.Fields{"appliance": name}
	hasRebooted := false
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				currentStatus <- "rebooting, waiting for appliance to come back online"
				hasRebooted = true
			} else if hasRebooted {
				currentStatus <- "switching partition"
			} else {
				currentStatus <- "installing"
			}
			return err
		}
		var s string
		details := status.GetDetails()
		if v, ok := status.GetStatusOk(); ok {
			s = *v
			currentStatus <- s
			if s == UpgradeStatusFailed && !util.InSlice(s, desiredStatuses) {
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", name, details))
			}
		}
		fields["image"] = details
		fields["status"] = s
		log.WithFields(fields).Infof("waiting for '%s' state", desiredStatuses)
		if util.InSlice(s, desiredStatuses) && s != UpgradeStatusFailed {
			return nil
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

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string) backoff.Operation {
	name := appliance.GetName()
	fields := log.Fields{"appliance": name}
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
			if s == UpgradeStatusFailed && !util.InSlice(s, desiredStatuses) {
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", name, status.GetDetails()))
			}
		}
		details := status.GetDetails()
		fields["image"] = details
		fields["status"] = s
		log.WithFields(fields).Infof("waiting for '%s' state", desiredStatuses)
		if util.InSlice(s, desiredStatuses) && s != UpgradeStatusFailed {
			return nil
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

func (u *UpgradeStatus) Subscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, current chan<- string) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	if err := backoff.Retry(u.upgradeStatusSubscribe(ctx, appliance, desiredStatuses, current), b); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
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

type WaitForApplianceStatus interface {
	WaitForState(ctx context.Context, appliance openapi.Appliance, expectedState string, status chan<- string) error
	// WaitForStatus tries appliance stats until the appliance has want status or it reaches the timeout
	WaitForStatus(ctx context.Context, appliance openapi.Appliance, want []string) error
}

type ApplianceStatus struct {
	Appliance *Appliance
}

func (u *ApplianceStatus) WaitForStatus(ctx context.Context, appliance openapi.Appliance, want []string) error {
	return backoff.Retry(func() error {
		stats, _, err := u.Appliance.Stats(ctx)
		if err != nil {
			return err
		}
		for _, stat := range stats.GetData() {
			if stat.GetId() == appliance.GetId() {
				if !util.InSlice(stat.GetStatus(), want) {
					return fmt.Errorf("Want status %s, got %s", want, stat.GetStatus())
				}
			}
		}
		return nil

	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
}

func (u *ApplianceStatus) WaitForState(ctx context.Context, appliance openapi.Appliance, expectedState string, status chan<- string) error {
	b := backoff.WithContext(&backoff.ExponentialBackOff{
		InitialInterval:     10 * time.Second,
		RandomizationFactor: 0.7,
		Multiplier:          2,
		MaxInterval:         20 * time.Second,
		MaxElapsedTime:      10 * time.Minute,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}, ctx)
	// initial sleep period
	time.Sleep(2 * time.Second)
	return backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		stats, _, err := u.Appliance.Stats(ctx)
		if err != nil {
			return err
		}

		for _, stat := range stats.GetData() {
			if stat.GetId() == appliance.GetId() {
				state := stat.GetState()
				fields := log.Fields{
					"status":        stat.GetStatus(),
					"current_state": state,
					"appliance":     stat.GetName(),
				}
				log.WithFields(fields).Infof(
					"Waiting for state %q",
					expectedState,
				)
				if status != nil {
					status <- state
				}
				if state != expectedState {
					log.WithFields(fields).Errorf("never reached desired state")
					return fmt.Errorf("never reached desired state %s", expectedState)
				}
			}
		}
		log.WithFields(log.Fields{
			"appliance":     appliance.GetName(),
			"current_state": expectedState,
		}).Info("reached desired state")
		return nil
	}, b)
}

func (u *UpgradeStatus) Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endState string, failState string, current <-chan string) {
	go func() {
		log.WithField("appliance", appliance.GetName()).Debug("Watching for appliance state")
		endMsg := "completed"
		previous := ""
		name := appliance.GetName()
		spinner := util.AddDefaultSpinner(p, name, "", endMsg)
		for status := range current {
			switch status {
			case endState:
				spinner.Increment()
			case failState:
				if len(previous) > 0 {
					spinner.Abort(false)
				} else if len(status) > 0 && status != previous {
					old := spinner
					spinner = util.AddDefaultSpinner(p, name, strings.ReplaceAll(status, "_", " "), endMsg, mpb.BarQueueAfter(old, false))
					old.Increment()
					previous = status
				}
			default:
				if len(status) > 0 && status != previous {
					old := spinner
					spinner = util.AddDefaultSpinner(p, name, strings.ReplaceAll(status, "_", " "), endMsg, mpb.BarQueueAfter(old, false))
					old.Increment()
					previous = status
				}
			}
		}
	}()
}
