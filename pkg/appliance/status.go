package appliance

import (
	"context"
	"fmt"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v7"
)

type WaitForUpgradeStatus interface {
	Wait(ctx context.Context, appliance openapi.Appliance, desiredStatus string, current chan<- string) error
	Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endState string, current <-chan string)
}

type UpgradeStatus struct {
	Appliance *Appliance
}

var defaultExponentialBackOff = &backoff.ExponentialBackOff{
	InitialInterval:     1 * time.Second,
	Multiplier:          2,
	RandomizationFactor: 0.7,
	MaxInterval:         10 * time.Second,
	MaxElapsedTime:      10 * time.Minute,
	Stop:                backoff.Stop,
	Clock:               backoff.SystemClock,
}

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatus string, currentStatus chan<- string) backoff.Operation {
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
			currentStatus <- s
			if status.GetStatus() == UpgradeStatusFailed {
				return backoff.Permanent(fmt.Errorf("Upgraded failed on %s - %s", appliance.GetName(), status.GetDetails()))
			}
		}
		fields["image"] = status.GetDetails()
		fields["status"] = s
		log.WithFields(fields).Infof("waiting for '%s' state", desiredStatus)
		if s == desiredStatus {
			return nil
		}
		return fmt.Errorf(
			"%s never reached %s, got %q %s",
			appliance.GetName(),
			desiredStatus,
			s,
			status.GetDetails(),
		)
	}
}

func (u *UpgradeStatus) Wait(ctx context.Context, appliance openapi.Appliance, desiredStatus string, current chan<- string) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	if err := backoff.Retry(u.upgradeStatus(ctx, appliance, desiredStatus, current), b); err != nil {
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
	WaitForState(ctx context.Context, appliance openapi.Appliance, expectedState string) error
}

type ApplianceStatus struct {
	Appliance *Appliance
}

func (u *ApplianceStatus) WaitForState(ctx context.Context, appliance openapi.Appliance, expectedState string) error {
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
					state,
				)
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

func (u *UpgradeStatus) Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endState string, current <-chan string) {
	go func() {
		endMsg := "completed"
		previous := ""
		name := appliance.GetName()
		spinner := util.AddDefaultSpinner(p, name, "", endMsg)
		for status := range current {
			if status != previous {
				spinner.Increment()
				old := spinner
				spinner = util.AddDefaultSpinner(p, name, status, endMsg, mpb.BarQueueAfter(old, false))
				if status == endState {
					spinner.Increment()
				}
				previous = status
			}
		}
	}()
}
