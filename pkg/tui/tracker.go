package tui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

type Tracker struct {
	mu           sync.Mutex
	container    *Progress
	appliance    openapi.Appliance
	bar          *mpb.Bar
	success      bool
	endMsg       string
	current      string
	statusReport <-chan string
}

// Watch initiates the progress tracking for each appliance update
// It succeeds if the status written to the statusReport is in the until slice
// It will abort the tracker if status the same as included in the failOn slice
func (t *Tracker) Watch(endMsg string, until, failOn []string) {
	t.endMsg = endMsg

	t.mu.Lock()
	t.bar = t.container.pc.New(1,
		mpb.SpinnerStyle(SpinnerStyle...),
		mpb.AppendDecorators(t.decoratorFunc(t.appliance.GetName())),
		mpb.BarFillerMiddleware(t.barFillerFunc()),
	)
	t.mu.Unlock()

	// failCount keeps track of the amount of failed status updates
	// the status needs to be reported as failed at least twice before
	// failing the spinner. This avoids pre-mature spinner abort when
	// the initial status is also part of the failOn status
	failCount := 0
	var ok bool
	for !t.bar.Completed() && !t.bar.Aborted() {
		if util.InSlice(t.current, until) {
			t.success = true
			t.complete()
		}
		if util.InSlice(t.current, failOn) {
			if failCount > 0 {
				t.complete()
			}
			failCount++
		}

		// This will keep the loop updating even if no new status is sent
		// if the parent context is done, it will abort the tracker
		select {
		case t.current, ok = <-t.statusReport:
			if !ok {
				t.abort()
			}
		case <-t.container.ctx.Done():
			t.abort()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (t *Tracker) complete() {
	for !t.bar.Completed() {
		t.bar.Increment()
	}
	t.bar.Wait()
}

func (t *Tracker) abort() {
	for !t.bar.Aborted() {
		t.bar.Abort(false)
	}
	t.bar.Wait()
}

func (t *Tracker) decoratorFunc(name string) decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		t.mu.Lock()
		defer t.mu.Unlock()
		if s.Completed && t.success {
			t.current = t.endMsg
		}
		if (s.Completed && !t.success) || s.Aborted {
			t.current = "failed"
		}
		return fmt.Sprintf("%s: %s", name, strings.ReplaceAll(t.current, "_", " "))
	})
}

func (t *Tracker) barFillerFunc() func(mpb.BarFiller) mpb.BarFiller {
	return func(bf mpb.BarFiller) mpb.BarFiller {
		return mpb.BarFillerFunc(func(w io.Writer, reqWidth int, st decor.Statistics) {
			t.mu.Lock()
			defer t.mu.Unlock()
			if st.Completed && t.success {
				io.WriteString(w, SpinnerDone)
				return
			}
			if (st.Completed && !t.success) || st.Aborted {
				io.WriteString(w, SpinnerErr)
				return
			}
			bf.Fill(w, reqWidth, st)
		})
	}
}
