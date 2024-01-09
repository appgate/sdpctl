package tui

import (
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func AddDefaultSpinner(p *mpb.Progress, name string, stage string, cmsg string, opts ...mpb.BarOption) *mpb.Bar {
	options := []mpb.BarOption{
		mpb.BarFillerOnComplete(Check),
		mpb.BarWidth(1),
		mpb.AppendDecorators(
			decor.Name(name, decor.WCSyncWidthR),
			decor.Name(":", decor.WC{W: 2, C: decor.WCSyncSpaceR.W}),
			decor.OnComplete(decor.OnAbort(decor.Name(stage), ""), cmsg),
		),
	}
	options = append(options, opts...)

	return p.New(1, mpb.SpinnerStyle(SpinnerStyle...), options...)
}
