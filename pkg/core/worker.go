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
	StartTime int64
	Abort     <-chan struct{}
	Input     InputChannel
	Output    Output
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
			curTime := time.Now().UnixNano()
			offset := curTime - w.StartTime
			if offset < item.TimeOffset {
				delay := item.TimeOffset - offset
				log.Debugf("[%s] Sleeping: %dns", w.Name, delay)
				w.Output.IncSleeping()
				time.Sleep(time.Duration(delay))
				w.Output.DecSleeping()
			}

			w.Output.IncBusy()
			res := w.Nib.Punch(item.Payload)
			w.Output.Push(res)
			w.Output.DecBusy()
			w.Output.DecWorking()
		}
	}
}

func NewBasicWorker(name string, abort chan struct{}, input InputChannel, output Output, startTime int64, nib Nib) Worker {
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
