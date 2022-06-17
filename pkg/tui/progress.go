package tui

import (
	"context"
	"io"
	"time"

	"github.com/vbauerster/mpb/v7"
)

type Progress struct {
	ctx      context.Context
	pc       *mpb.Progress
	trackers []*Tracker
}

// NewProgress initiates a new progress tracking container
func NewProgress(ctx context.Context, out io.Writer, options ...mpb.ContainerOption) *Progress {
	p := Progress{
		ctx: ctx,
	}
	options = append(options, mpb.WithWidth(1), mpb.WithOutput(out))
	p.pc = mpb.NewWithContext(ctx, options...)
	return &p
}

// AddTracker will add a status tracker to the appliance
// the returned channel is to be used by other functions to update the tracker progress
func (p *Progress) AddTracker(name, endMsg string) (*Tracker, chan<- string) {
	t := Tracker{
		container:    p,
		name:         name,
		statusReport: make(chan string, 1),
		msgProxy:     make(chan string),
	}

	t.mu.Lock()
	t.bar = t.container.pc.New(1,
		mpb.SpinnerStyle(SpinnerStyle...),
		mpb.AppendDecorators(t.decoratorFunc(t.name, endMsg)),
		mpb.BarFillerMiddleware(t.barFillerFunc()),
	)
	t.mu.Unlock()

	p.trackers = append(p.trackers, &t)
	return &t, t.statusReport
}

// Complete will complete all currently active trackers and wait for them to finish before returning
func (p *Progress) Complete() {
	for _, t := range p.trackers {
		t.complete()
	}
}

// Abort will abort all currently active trackers and wait for them to finish before returning
func (p *Progress) Abort() {
	for _, t := range p.trackers {
		t.abort(false)
	}
}

// Wait will wait for all progress bars to complete with a timeout
// If deadline is reached before the bars are complete, it will abort
// all bars remaining and return
func (p *Progress) Wait(timeout time.Duration) {
	done := make(chan bool)
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	go func() {
		p.pc.Wait()
		done <- true
		close(done)
	}()

	select {
	case <-ctx.Done():
		p.Abort()
	case <-done:
	}
}
