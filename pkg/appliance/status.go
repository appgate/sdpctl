package appliance

import (
	"context"
	"fmt"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
)

type WaitForApplianceStatus interface {
	WaitForApplianceState(ctx context.Context, appliance openapi.Appliance, want []string, status chan<- string) error
	// WaitForStatus tries appliance stats until the appliance has want status or it reaches the timeout
	WaitForApplianceStatus(ctx context.Context, appliance openapi.Appliance, want []string, status chan<- string) error
}

type ApplianceStatus struct {
	Appliance *Appliance
}

func (u *ApplianceStatus) WaitForApplianceStatus(ctx context.Context, appliance openapi.Appliance, want []string, status chan<- string) error {
	logEntry := log.WithFields(log.Fields{
		"appliance": appliance.GetName(),
	})
	logEntry.WithField("want", want).Info("polling for appliance status")
	return backoff.Retry(func() error {
		stats, _, err := u.Appliance.Stats(ctx)
		if err != nil {
			return err
		}
		for _, stat := range stats.GetData() {
			if stat.GetId() == appliance.GetId() {
				current := stat.GetStatus()
				logEntry.WithFields(log.Fields{
					"current": current,
				}).Debug("recieved appliance status")
				if status != nil {
					status <- current
				}
				if !util.InSlice(current, want) {
					return fmt.Errorf("Want status %s, got %s", want, current)
				}
			}
		}
		logEntry.Info("reached wanted appliance status")
		return nil

	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
}

func (u *ApplianceStatus) WaitForApplianceState(ctx context.Context, appliance openapi.Appliance, want []string, status chan<- string) error {
	b := backoff.WithContext(&backoff.ExponentialBackOff{
		InitialInterval:     10 * time.Second,
		RandomizationFactor: 0.7,
		Multiplier:          2,
		MaxInterval:         20 * time.Second,
		MaxElapsedTime:      10 * time.Minute,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}, ctx)
	logEntry := log.WithFields(log.Fields{
		"appliance": appliance.GetName(),
	})
	logEntry.WithField("want", want).Info("polling for appliance state")
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
					"current": state,
				}
				logEntry.WithFields(fields).Debug("recieved appliance state")
				if status != nil {
					status <- state
				}
				if !util.InSlice(state, want) {
					return fmt.Errorf("never reached desired state %s", want)
				}
			}
		}
		logEntry.Info("reached wanted appliance state")
		return nil
	}, b)
}
