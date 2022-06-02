package appliance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
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
	Wait(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string) error
	// Subscribe does expodential backoff retries on upgrade status until it reaches a desiredStatuses and reports it to current <- string
	Subscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, current chan<- string) error
	// Watch will print the spinner progress bar and listen for message from <-current and present it to the statusbar
	Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endState string, failState string, current <-chan string)
}

type UpgradeStatus struct {
	Appliance *Appliance
	mu        sync.Mutex
}

func (u *UpgradeStatus) Wait(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	if err := backoff.Retry(u.upgradeStatus(ctx, appliance, desiredStatuses, undesiredStatuses), b); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string) backoff.Operation {
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
			if util.InSlice(s, undesiredStatuses) {
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", name, status.GetDetails()))
			}
		}
		details := status.GetDetails()
		fields["image"] = details
		fields["status"] = s
		log.WithFields(fields).Infof("waiting for '%s' state", desiredStatuses)
		if util.InSlice(s, desiredStatuses) {
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

func (u *UpgradeStatus) upgradeStatusSubscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, currentStatus chan<- string) backoff.Operation {
	name := appliance.GetName()
	fields := log.Fields{"appliance": name}
	hasRebooted := false
	isInitialState := false
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
			if !isInitialState {
				isInitialState = true
				return errors.New("initial state throwaway")
			}
			currentStatus <- s
			if util.InSlice(s, undesiredStatuses) {
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

func (u *UpgradeStatus) Subscribe(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, current chan<- string) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return backoff.Retry(u.upgradeStatusSubscribe(ctx, appliance, desiredStatuses, undesiredStatuses, current), b)
	}
}

// Watch will print the spinner progress bar and listen for message from <-current and present it to the statusbar
func (u *UpgradeStatus) Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endState string, failState string, current <-chan string) {
	name := appliance.GetName()
	// we will lock each time we add a new bar to avoid duplicates
	u.mu.Lock()
	barMessage := make(chan string, 1)
	bar := p.New(1,
		mpb.SpinnerStyle(prompt.SpinnerStyle...),
		mpb.AppendDecorators(
			func(cond string) decor.Decorator {
				var (
					done       = false
					defaultStr = cond
					doneText   string
					body       string
				)
				return decor.Any(func(s decor.Statistics) string {
					if done {
						return doneText
					}
					if s.Completed || ctx.Err() != nil {
						done = true
						if ctx.Err() == nil {
							doneText = cond
						} else {
							doneText = fmt.Sprintf("time out on %s %s", cond, ctx.Err())
						}
						return doneText
					}
					select {
					case msg := <-barMessage:
						body = fmt.Sprintf("%s %s", cond, msg)
						// default refreshrate is 150ms, we will check
						// the channel every 2/3 of that time to reduce duplicate
					case <-time.After(100 * time.Millisecond):
						if len(body) == 0 {
							body = defaultStr
						}
					}
					return body
				}, decor.WCSyncSpaceR)
			}(name),
		),
		mpb.BarFillerMiddleware(
			prompt.CheckBarFiller(ctx, func(c context.Context) bool {
				return ctx.Err() == nil
			}),
		),
		mpb.BarWidth(1),
	)

	go func() {
		<-ctx.Done()
		if !bar.Completed() {
			bar.Increment()
		}
	}()

	u.mu.Unlock()
	// Proxy message from <-current to <-barMessage channel,
	// each time we update it, we will lock to avoid duplicate
	for !bar.Completed() {
		v, ok := <-current
		if !ok {
			bar.Abort(true)
			break
		}

		go func() {
			u.mu.Lock()
			barMessage <- v
			u.mu.Unlock()
		}()

		if v == failState {
			bar.Abort(false)
			break
		}
		if v == endState {
			go func() {
				u.mu.Lock()
				bar.Increment()
				u.mu.Unlock()
			}()
			break
		}
	}
	if !bar.Completed() {
		bar.Increment()
	}
}
