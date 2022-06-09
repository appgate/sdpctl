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
	// WaitForUpgradeStatus does expodential backoff retries on upgrade status until it reaches a desiredStatuses and reports it to current <- string
	WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, current chan<- string) error
	// Watch will print the spinner progress bar and listen for message from <-current and present it to the statusbar
	Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endStates []string, failStates []string, current <-chan string)
}

type UpgradeStatus struct {
	Appliance *Appliance
	mu        sync.Mutex
}

func (u *UpgradeStatus) upgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, currentStatus chan<- string) backoff.Operation {
	name := appliance.GetName()
	logEntry := log.WithField("appliance", name)
	logEntry.WithField("want", desiredStatuses).Info("polling for upgrade status")
	hasRebooted := false
	isInitialState := false
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		status, err := u.Appliance.UpgradeStatus(ctx, appliance.GetId())
		if err != nil {
			if currentStatus != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					currentStatus <- "rebooting, waiting for appliance to come back online"
					hasRebooted = true
				} else if hasRebooted {
					currentStatus <- "switching partition"
				} else {
					currentStatus <- "installing"
				}
			}
			logEntry.WithError(err).Debug("appliance unreachable")
			return err
		}
		var s string
		details := status.GetDetails()
		if v, ok := status.GetStatusOk(); ok {
			s = *v
			logEntry.WithField("current", s).Debug("recieved upgrade status")
			if !isInitialState {
				isInitialState = true
				return errors.New("initial state throwaway")
			}
			if currentStatus != nil {
				currentStatus <- s
			}
			if util.InSlice(s, undesiredStatuses) {
				return backoff.Permanent(fmt.Errorf("Upgrade failed on %s - %s", name, details))
			}
		}
		if util.InSlice(s, desiredStatuses) {
			logEntry.Info("reached wanted upgrade status")
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

func (u *UpgradeStatus) WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, current chan<- string) error {
	b := backoff.WithContext(defaultExponentialBackOff, ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return backoff.Retry(u.upgradeStatus(ctx, appliance, desiredStatuses, undesiredStatuses, current), b)
	}
}

// Watch will print the spinner progress bar and listen for message from <-current and present it to the statusbar
func (u *UpgradeStatus) Watch(ctx context.Context, p *mpb.Progress, appliance openapi.Appliance, endStates []string, failStates []string, current <-chan string) {
	name := appliance.GetName()
	logEntry := log.WithField("appliance", name)
	logEntry.Info("watching status on appliance")
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
							doneText = fmt.Sprintf("time out on %s: %s", cond, ctx.Err())
						}
						return doneText
					}
					select {
					case msg := <-barMessage:
						body = fmt.Sprintf("%s: %s", cond, strings.ReplaceAll(msg, "_", " "))
						// default refreshrate is 150ms, we will check
						// the channel every 2/3 of that time to reduce duplicate
					case <-time.After(100 * time.Millisecond):
						if len(body) == 0 {
							body = defaultStr
						}
					}
					return body
				})
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

		if util.InSlice(v, failStates) {
			bar.Abort(false)
			break
		}
		if util.InSlice(v, endStates) {
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
