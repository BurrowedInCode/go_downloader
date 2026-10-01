package main

import (
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type mpbAdapter struct {
	progress *mpb.Progress
	bar      *mpb.Bar
}

func (a *mpbAdapter) SetTotal(total int64) {
	if total < 0 {
		a.bar = a.progress.AddSpinner(0, mpb.PrependDecorators(decor.CurrentKibiByte("% .1f")))
	} else {
		a.bar = a.progress.AddBar(total,
			mpb.PrependDecorators(decor.Counters(decor.SizeB1024(0), "% .1f / % .1f")),
			mpb.AppendDecorators(decor.Percentage()))
	}
}

func (a *mpbAdapter) Add(bytes int64) {
	a.bar.IncrInt64(bytes)
}

func (a *mpbAdapter) Abort() {
	if a.bar == nil {
		return
	}
	a.bar.Abort(false)
}

func (a *mpbAdapter) Complete() {
	a.bar.SetTotal(-1, true)
}
