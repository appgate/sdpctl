package appliance

import (
	"context"
	"fmt"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
)

type WaitForUpgradeStatus interface {
	Wait(ctx context.Context, appliances []openapi.Appliance, desiredStatus string) error
}

type UpgradeStatus struct {
	Appliance *Appliance
}

var defaultExponentialBackOff = &backoff.ExponentialBackOff{
	InitialInterval: 10 * time.Second,
	Multiplier:      1,
	MaxInterval:     1 * time.Minute,
	MaxElapsedTime:  10 * time.Minute,
	Stop:            backoff.Stop,
	Clock:           backoff.SystemClock,
}

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatus string) backoff.Operation {
	fields := log.Fields{"appliance": appliance.GetName()}
	return func() error {
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			return err
		}
		var s string
		if v, ok := status.GetStatusOk(); ok {
			s = *v
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

func (u *UpgradeStatus) Wait(ctx context.Context, appliances []openapi.Appliance, desiredStatus string) error {
	for _, i := range appliances {
		b := backoff.WithContext(defaultExponentialBackOff, ctx)
		if err := backoff.Retry(u.upgradeStatus(ctx, i, desiredStatus), b); err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

type WaitForApplianceStatus interface {
	WaitForState(ctx context.Context, appliances []openapi.Appliance, expectedState string) error
}

type ApplianceStatus struct {
	Appliance *Appliance
}

func (u *ApplianceStatus) WaitForState(ctx context.Context, appliances []openapi.Appliance, expectedState string) error {
	b := backoff.WithContext(&backoff.ExponentialBackOff{
		InitialInterval:     10 * time.Second,
		RandomizationFactor: 0.7,
		Multiplier:          2,
		MaxInterval:         20 * time.Second,
		MaxElapsedTime:      20 * time.Minute,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}, ctx)
	// initial sleep period
	time.Sleep(5 * time.Second)
	return backoff.Retry(func() error {
		stats, _, err := u.Appliance.Stats(ctx)
		if err != nil {
			return err
		}
		result := make(map[string]int)
		candidates := make([]openapi.StatsAppliancesListAllOfData, 0)

		for _, stat := range stats.GetData() {
			for _, appliance := range appliances {
				if stat.GetId() == appliance.GetId() {
					candidates = append(candidates, stat)
				}
			}
		}
		for _, stat := range candidates {
			fields := log.Fields{
				"status":        stat.GetStatus(),
				"current_state": stat.GetState(),
				"appliance":     stat.GetName(),
			}
			log.WithFields(fields).Infof(
				"Waiting for state %q",
				stat.GetState(),
			)
			if stat.GetState() == expectedState {
				result[stat.GetId()] = 1
			}
		}
		if len(result) == len(appliances) {
			log.Infof("reached desired %q on %d appliances", expectedState, len(appliances))
			return nil
		}
		return fmt.Errorf("never reached desired state %s", expectedState)
	}, b)
}
