package tui

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/appgate/sdpctl/pkg/util"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
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
	failReason   chan string
}

// Watch initiates the progress tracking for each appliance update
// It succeeds if the status written to the statusReport is in the until slice
// It will abort the tracker if status the same as included in the failOn slice
func (t *Tracker) Watch(until, failOn []string) {
	defer func() {
		t.done = true
		if !t.bar.Completed() || !t.bar.Aborted() {
			t.abort(false)
		}
	}()

	var (
		msg string
		ok  bool
	)
	for {
		msg, ok = <-t.statusReport
		if !ok {
			t.abort(false)
			break
		}
		t.mu.Lock()
		msg = strings.TrimSpace(msg)
		t.current = msg
		t.mu.Unlock()
		if util.InSlice(msg, until) {
			t.current = t.endMsg
			t.success = true
			t.complete()
			break
		}
		if util.InSlice(msg, failOn) {
			// on failure, we expect to get the reason from this channel
			reason, ok := <-t.failReason
			if ok {
				// Tracker does not deal well with multiple lines, so we need some string sanitation
				// Just grab the first line and trim trailing special chars, such as ':'
				firstLine := strings.Split(reason, "\n")[0]
				re := regexp.MustCompile(`[^\w]$`)
				show := re.ReplaceAllString(firstLine, "")
				t.mu.Lock()
				t.current = show
				t.mu.Unlock()
			}
			t.complete()
			break
		}
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
		return mpb.BarFillerFunc(func(w io.Writer, st decor.Statistics) error {
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.done {
				if t.success {
					io.WriteString(w, Check)
					return nil
				}
				io.WriteString(w, Cross)
				return nil
			}
			return bf.Fill(w, st)
		})
	}
}

// Update is used to send strings for the tracker to present in the progress tracking
func (t *Tracker) Update(s string) {
	select {
	case t.statusReport <- s:
		// wait one refresh cycle to give bars a chance to update properly
		time.Sleep(defaultRefreshRate)
	default:
	}
}

func (t *Tracker) Fail(s string) {
	select {
	case t.failReason <- s:
		// wait one refresh cycle to give bars a chance to update properly
		time.Sleep(defaultRefreshRate)
	default:
	}
}

func (t *Tracker) Current() string {
	return t.current
}
