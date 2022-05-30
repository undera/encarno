package core

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// basic worker and regex-capable worker, regex-capable should read file on its own
// track expected request time and factual, report own overloaded state, auto-stop if unable to conform

type Worker interface {
	Run()
}

type BasicWorker struct {
	Name      string
	Nib       Nib
	StartTime time.Time
	Abort     <-chan struct{}
	Input     InputChannel
	Output    Output
	values    map[string][]byte
}

func (w *BasicWorker) Run() {
outer:
	for {
		select {
		case <-w.Abort:
			log.Debugf("Aborting worker: %s", w.Name)
			break outer
		default:
			item := <-w.Input

			w.Output.IncWorking()
			item.ReplaceValues(w.values)

			curTime := time.Now()
			offset := curTime.Sub(w.StartTime)
			if offset < item.TimeOffset {
				delay := item.TimeOffset - offset
				log.Debugf("[%s] Sleeping: %dns", w.Name, delay)
				w.Output.IncSleeping()
				time.Sleep(delay)
				w.Output.DecSleeping()
			}

			w.Output.IncBusy()
			res := w.Nib.Punch(item.Payload)
			res.StartDivergence = res.StartTime.Sub(w.StartTime.Add(item.TimeOffset))
			res.ExtractValues(item.RegexOut, w.values)
			w.Output.Push(res)
			w.Output.DecBusy()
			w.Output.DecWorking()
		}
	}
}

func NewBasicWorker(name string, abort chan struct{}, input InputChannel, output Output, startTime time.Time, nib Nib) Worker {
	var b Worker
	b = &BasicWorker{
		Name:      name,
		Nib:       nib,
		Abort:     abort,
		Input:     input,
		Output:    output,
		StartTime: startTime,
	}
	return b
}
