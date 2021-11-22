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
	Wait(ctx context.Context, appliances []openapi.Appliance) error
}

type UpgradeStatus struct {
	Appliance *Appliance
}

func (u *UpgradeStatus) Wait(ctx context.Context, appliances []openapi.Appliance) error {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     10 * time.Second,
		RandomizationFactor: 0.7,
		Multiplier:          2,
		MaxInterval:         5 * time.Minute,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
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
					log.WithFields(fields).Infof("upgrade status %q %s", s, status.GetDetails())
				}
				if s == UpgradeStatusReady {
					return nil
				}
				return fmt.Errorf(
					"%s never reached %s, got %q %s",
					i.GetName(),
					UpgradeStatusReady,
					s,
					status.GetDetails(),
				)
			}, b)
		})
	}
	return g.Wait()
}
