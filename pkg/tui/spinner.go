package tui

import (
	"context"
	"io"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

func AddDefaultSpinner(p *mpb.Progress, name string, stage string, cmsg string, opts ...mpb.BarOption) *mpb.Bar {
	options := []mpb.BarOption{
		mpb.BarFillerOnComplete(SpinnerDone),
		mpb.BarWidth(1),
		mpb.AppendDecorators(
			decor.Name(name, decor.WCSyncWidthR),
			decor.Name(":", decor.WC{W: 2, C: decor.DidentRight}),
			decor.OnComplete(decor.OnAbort(decor.Name(stage), ""), cmsg),
		),
	}
	options = append(options, opts...)

	return p.New(1, mpb.SpinnerStyle(SpinnerStyle...), options...)
}

func CheckBarFiller(
	waitCtx context.Context,
	success func(context.Context) bool,
) func(mpb.BarFiller) mpb.BarFiller {
	return func(base mpb.BarFiller) mpb.BarFiller {
		done := false
		var doneText string
		return mpb.BarFillerFunc(func(w io.Writer, reqWidth int, st decor.Statistics) {
			if done {
				io.WriteString(w, doneText)
				return
			}

			if st.Completed || waitCtx.Err() != nil {
				done = true
				if success(waitCtx) {
					doneText = SpinnerDone
				} else {
					doneText = SpinnerErr
				}
				io.WriteString(w, doneText)
			} else {
				base.Fill(w, reqWidth, st)
			}

		})
	}
}
