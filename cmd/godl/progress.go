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
	bar := a.progress.AddBar(total,
		mpb.PrependDecorators(decor.Counters(decor.SizeB1024(0), "% .1f / % .1f")),
		mpb.AppendDecorators(decor.Percentage()))
	a.bar = bar
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
