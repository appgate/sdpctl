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
	WaitForApplianceStatus(ctx context.Context, appliance openapi.Appliance, want []string) error
}

type ApplianceStatus struct {
	Appliance *Appliance
}

func (u *ApplianceStatus) WaitForApplianceStatus(ctx context.Context, appliance openapi.Appliance, want []string) error {
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
					want,
				)
				if status != nil {
					status <- state
				}
				if !util.InSlice(state, want) {
					log.WithFields(fields).Errorf("never reached desired state")
					return fmt.Errorf("never reached desired state %s", want)
				}
			}
		}
		log.WithFields(log.Fields{
			"appliance":     appliance.GetName(),
			"current_state": want,
		}).Info("reached desired state")
		return nil
	}, b)
}
