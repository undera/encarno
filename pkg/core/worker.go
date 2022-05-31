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
	Name           string
	Nib            Nib
	StartTime      time.Time
	Abort          <-chan struct{}
	Input          InputChannel
	Output         Output
	values         map[string][]byte
	Finished       bool
	IterationCount int
}

func (w *BasicWorker) Run() {
	timeTracker := TimeTracker{Stamp: time.Now()}
outer:
	for {
		select {
		case <-w.Abort:
			log.Debugf("Aborting worker: %s", w.Name)
			break outer
		default:
			timeTracker.Switch()
			item := <-w.Input
			timeTracker.Switch()
			if item == nil {
				break outer
			}

			w.Output.IncWorking()
			w.IterationCount += 1
			item.ReplaceValues(w.values)

			expectedStart := w.StartTime.Add(item.TimeOffset)
			curTime := time.Now()
			delay := expectedStart.Sub(curTime)
			if delay > 0 {
				log.Debugf("[%s] Sleeping: %dns", w.Name, delay)
				w.Output.IncSleeping()
				timeTracker.Switch()
				time.Sleep(delay)
				timeTracker.Switch()
				w.Output.DecSleeping()
			}

			w.Output.IncBusy()
			res := w.Nib.Punch(item.Payload)
			res.StartMissed = res.StartTime.Sub(expectedStart)
			res.ExtractValues(item.RegexOut, w.values)
			w.Output.Push(res)
			w.Output.DecBusy()
			w.Output.DecWorking()
		}
	}
	log.Debugf("Worker finished: %s", w.Name)
	w.Finished = true
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

type TimeTracker struct {
	Stamp         time.Time
	IsSleeping    bool
	SpentWorking  time.Duration
	SpentSleeping time.Duration
}

func (t *TimeTracker) Switch() {
	if t.IsSleeping {
		t.SpentSleeping += time.Now().Sub(t.Stamp)
	} else {
		t.SpentWorking += time.Now().Sub(t.Stamp)
	}
	t.IsSleeping = !t.IsSleeping
	t.Stamp = time.Now()
}
