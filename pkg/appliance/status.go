package appliance

import (
	"context"
	"fmt"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
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

func (u *UpgradeStatus) Wait(ctx context.Context, appliances []openapi.Appliance, desiredStatus string) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, appliance := range appliances {
		i := appliance
		g.Go(func() error {
			fields := log.Fields{"appliance": i.GetName()}
			return backoff.Retry(func() error {
				status, err := u.Appliance.UpgradeStatus(ctx, i.GetId())
				if err != nil {
					return err
				}
				var s string
				if v, ok := status.GetStatusOk(); ok {
					s = *v
					if status.GetStatus() == UpgradeStatusFailed {
						log.WithFields(fields).Errorf(status.GetDetails())
						return backoff.Permanent(fmt.Errorf("Upgraded failed on %s - %s", i.GetName(), status.GetDetails()))
					}
					log.WithFields(fields).Infof("upgrade status %q %s waiting for %s", s, status.GetDetails(), desiredStatus)
				}

				if s == desiredStatus {
					return nil
				}
				return fmt.Errorf(
					"%s never reached %s, got %q %s",
					i.GetName(),
					desiredStatus,
					s,
					status.GetDetails(),
				)
			}, defaultExponentialBackOff)
		})
	}
	return g.Wait()
}

type WaitForApplianceStatus interface {
	WaitForState(ctx context.Context, appliances []openapi.Appliance, expectedState string) error
}

type ApplianceStatus struct {
	Appliance *Appliance
}

func (u *ApplianceStatus) WaitForState(ctx context.Context, appliances []openapi.Appliance, expectedState string) error {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     10 * time.Second,
		RandomizationFactor: 0.7,
		Multiplier:          2,
		MaxInterval:         5 * time.Minute,
		MaxElapsedTime:      10 * time.Minute,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	// inital sleep period
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
			fields := log.Fields{"appliance": stat.GetName()}
			log.WithFields(fields).Infof(
				"got status %s state %q expects %q",
				stat.GetStatus(),
				stat.GetState(),
				expectedState,
			)
			if stat.GetState() == expectedState {
				result[stat.GetId()] = 1
			}
		}
		if len(result) == len(appliances) {
			log.Infof("reached desired %q on %d appliances", expectedState, len(appliances))
			return nil
		}
		return fmt.Errorf("never reached expected state %q", expectedState)
	}, b)
}
