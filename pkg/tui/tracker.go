package tui

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/appgate/sdpctl/pkg/util"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

type Tracker struct {
	mu           sync.Mutex
	container    *Progress
	name         string
	bar          *mpb.Bar
	current      string
	endMsg       string
	success      bool
	done         bool
	statusReport chan string
}

// Watch initiates the progress tracking for each appliance update
// It succeeds if the status written to the statusReport is in the until slice
// It will abort the tracker if status the same as included in the failOn slice
func (t *Tracker) Watch(until, failOn []string) {
	defer func() {
		t.done = true
		close(t.statusReport)
		if !t.bar.Completed() || !t.bar.Aborted() {
			t.abort(false)
		}
	}()

	// failCount keeps track of the amount of failed status updates
	// the status needs to be reported as failed at least twice before
	// failing the spinner. This avoids pre-mature spinner abort when
	// the initial status is also part of the failOn status
	failCount := 0
	var msg string
	var ok bool
	for {
		if util.InSlice(msg, until) {
			t.current = t.endMsg
			t.success = true
			t.complete()
			break
		}
		if util.InSlice(msg, failOn) {
			if failCount > 0 {
				t.current = "failed"
				t.abort(false)
				break
			}
			failCount++
			continue
		}
		msg, ok = <-t.statusReport
		if !ok {
			t.abort(false)
			break
		}
		t.current = msg
	}
}

func (t *Tracker) complete() {
	t.bar.Increment()
}

func (t *Tracker) abort(drop bool) {
	t.bar.Abort(drop)
}

func (t *Tracker) decoratorFunc(name string) decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		t.mu.Lock()
		defer t.mu.Unlock()
		return fmt.Sprintf("%s: %s", name, strings.ReplaceAll(t.current, "_", " "))
	})
}

func (t *Tracker) barFillerFunc() func(mpb.BarFiller) mpb.BarFiller {
	return func(bf mpb.BarFiller) mpb.BarFiller {
		return mpb.BarFillerFunc(func(w io.Writer, reqWidth int, st decor.Statistics) {
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.done {
				if t.success {
					io.WriteString(w, SpinnerDone)
					return
				}
				io.WriteString(w, SpinnerErr)
				return
			}
			bf.Fill(w, reqWidth, st)
		})
	}
}
