package core

import (
	log "github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)

// basic worker and regex-capable worker, regex-capable should read file on its own
// track expected request time and factual, report own overloaded state, auto-stop if unable to conform

// TODO refine what's actually needed
type Status interface {
	DecBusy()
	IncBusy()
	IncSleeping()
	DecSleeping()
	IncWorking()
	DecWorking()
	GetWorking() int64
	GetSleeping() int64
	GetBusy() int64
}

type StatusImpl struct {
	sleeping int64
	busy     int64
	working  int64
}

func (o *StatusImpl) GetWorking() int64 {
	return o.working
}

func (o *StatusImpl) GetSleeping() int64 {
	return o.sleeping
}

func (o *StatusImpl) GetBusy() int64 {
	return o.busy
}

func (o *StatusImpl) IncWorking() {
	atomic.AddInt64(&o.working, 1)
}

func (o *StatusImpl) DecWorking() {
	atomic.AddInt64(&o.working, -1)
	if o.working < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *StatusImpl) IncSleeping() {
	atomic.AddInt64(&o.sleeping, 1)
	if o.sleeping < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *StatusImpl) DecSleeping() {
	atomic.AddInt64(&o.sleeping, -1)
}

func (o *StatusImpl) DecBusy() {
	atomic.AddInt64(&o.busy, -1)
	if o.busy < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *StatusImpl) IncBusy() {
	atomic.AddInt64(&o.busy, 1)
}

type Worker struct {
	Name           string
	Nib            Nib
	StartTime      time.Time
	Abort          <-chan struct{}
	Input          InputChannel
	Output         Output
	values         map[string][]byte
	Finished       bool
	IterationCount int
	Status         Status
}

func (w *Worker) Run() {
	timeTracker := TimeTracker{Stamp: time.Now()}
outer:
	for {
		// TODO: only measure single iteration with time tracker, record its ratio into result
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

			w.Status.IncWorking()
			w.IterationCount += 1
			item.ReplaceValues(w.values)

			expectedStart := w.StartTime.Add(item.TimeOffset)
			curTime := time.Now()
			delay := expectedStart.Sub(curTime)
			if delay > 0 {
				log.Debugf("[%s] Sleeping: %dns", w.Name, delay)
				w.Status.IncSleeping()
				timeTracker.Switch()
				time.Sleep(delay)
				timeTracker.Switch()
				w.Status.DecSleeping()
			}

			w.Status.IncBusy()
			res := w.Nib.Punch(item)
			res.StartMissed = res.StartTime.Sub(expectedStart)
			res.ExtractValues(item.RegexOut, w.values)
			w.Output.Push(res)
			w.Status.DecBusy()
			w.Status.DecWorking()
		}
	}
	log.Debugf("Worker finished: %s", w.Name)
	w.Finished = true
}

func NewBasicWorker(name string, abort chan struct{}, input InputChannel, output Output, startTime time.Time, nib Nib, status Status) *Worker {
	b := &Worker{
		Name:      name,
		Nib:       nib,
		Abort:     abort,
		Input:     input,
		Output:    output,
		StartTime: startTime,
		Status:    status,
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
	sub := time.Now().Sub(t.Stamp)
	//log.Debugf("Inc: %v %d", t.IsSleeping, sub)
	if t.IsSleeping {
		t.SpentSleeping += sub
	} else {
		t.SpentWorking += sub
	}
	t.IsSleeping = !t.IsSleeping
	t.Stamp = time.Now()
}

// something spawns workers on-demand - either on schedule, or to sustain hits/s
// each worker knows its input reader, nib (can be different nibs btw), output writer
// auto-USL scenario, scheduled scenario, external "stpd" scenario
// spit out some stats each N seconds

type WorkerSpawner interface {
	Run()
	Interrupt()
}
