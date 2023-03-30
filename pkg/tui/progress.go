package tui

import (
	"context"
	"io"
	"time"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

type Progress struct {
	ctx      context.Context
	pc       *mpb.Progress
	trackers []*Tracker
}

var (
	defaultRefreshRate = 120 * time.Millisecond
)

// New initiates a new progress tracking container
func New(ctx context.Context, out io.Writer, options ...mpb.ContainerOption) *Progress {
	p := Progress{
		ctx: ctx,
	}
	options = append(options, mpb.WithWidth(1), mpb.WithOutput(out), mpb.WithRefreshRate(defaultRefreshRate))
	p.pc = mpb.NewWithContext(ctx, options...)
	return &p
}

// AddTracker will add a status tracker to the appliance
// the returned channel is to be used by other functions to update the tracker progress
func (p *Progress) AddTracker(name, state, endMsg string, opts ...mpb.BarOption) *Tracker {
	t := Tracker{
		container:    p,
		name:         name,
		current:      state,
		endMsg:       endMsg,
		statusReport: make(chan string, 1),
		failReason:   make(chan string, 1),
	}

	t.mu.Lock()
	barOptions := []mpb.BarOption{
		mpb.AppendDecorators(t.decoratorFunc(t.name)),
		mpb.BarFillerMiddleware(t.barFillerFunc()),
	}
	barOptions = append(barOptions, opts...)
	t.bar = p.pc.New(1,
		mpb.SpinnerStyle(SpinnerStyle...),
		barOptions...,
	)
	t.mu.Unlock()

	p.trackers = append(p.trackers, &t)
	return &t
}

func (p *Progress) FileUploadProgress(name, endMsg string, size int64, reader io.Reader) (io.Reader, *Tracker) {
	bar := p.pc.AddBar(
		size,
		mpb.BarWidth(50),
		mpb.BarFillerOnComplete(endMsg),
		mpb.PrependDecorators(
			decor.Spinner(SpinnerStyle, decor.WC{W: 2}),
			decor.Name(name, decor.WC{W: len(name) + 1}),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.CountersKibiByte("% .2f / % .2f"), ""),
			decor.OnComplete(decor.Name(" | "), ""),
			decor.OnComplete(decor.AverageSpeed(decor.UnitKiB, "% .2f"), ""),
		),
	)

	t := Tracker{
		bar:          bar,
		container:    p,
		name:         name,
		current:      "waiting",
		endMsg:       endMsg,
		statusReport: make(chan string, 1),
		failReason:   make(chan string, 1),
	}
	p.trackers = append(p.trackers, &t)

	qt := p.AddTracker(name, "waiting for server ok", endMsg, mpb.BarQueueAfter(bar, true))

	return bar.ProxyReader(reader), qt
}

func (p *Progress) FileDownloadProgress(name, endMsg string, size int64, width int, reader io.Reader) io.ReadCloser {
	bar := p.pc.AddBar(
		size,
		mpb.BarWidth(width),
		mpb.BarFillerOnComplete(endMsg),
		mpb.PrependDecorators(
			decor.OnComplete(decor.Spinner(SpinnerStyle, decor.WC{W: 2}), Check),
			decor.Name(name, decor.WC{W: len(name) + 1}),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.CountersKibiByte("% .2f / % .2f"), ""),
			decor.OnComplete(decor.Name(" | "), ""),
			decor.OnComplete(decor.AverageSpeed(decor.UnitKiB, "% .2f"), ""),
		),
	)
	return bar.ProxyReader(reader)
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
func (p *Progress) Wait() {
	done := make(chan bool)
	ctx, cancel := context.WithTimeout(p.ctx, 2*defaultRefreshRate)
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
