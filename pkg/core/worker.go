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
	InputPayload   InputChannel
	InputSchedule  ScheduleChannel
	Output         Output
	Values         map[string][]byte
	Finished       bool
	IterationCount int
	Status         Status
}

func (w *Worker) Run() {
	timeTracker := TimeTracker{}
outer:
	for {
		// TODO: only measure single iteration with time tracker, record its ratio into result
		select {
		case <-w.Abort:
			log.Debugf("Aborting worker: %s", w.Name)
			break outer
		default:
			shouldStop := w.Iteration(&timeTracker)
			if shouldStop {
				break outer
			}
		}
	}
	log.Debugf("Worker finished: %s", w.Name)
	w.Finished = true
	// TODO: somehow notify workers array/count
}

func (w *Worker) Iteration(timeTracker *TimeTracker) bool {
	timeTracker.Reset()
	timeTracker.Sleeping()
	offset := <-w.InputSchedule
	timeTracker.Working()

	w.Status.IncWorking()
	w.IterationCount += 1

	item := <-w.InputPayload
	if item == nil {
		return true
	}

	item.ReplaceValues(w.Values)

	expectedStart := w.StartTime.Add(offset)
	delay := expectedStart.Sub(time.Now())
	if delay > 0 {
		log.Debugf("[%s] Sleeping: %dns", w.Name, delay)
		w.Status.IncSleeping()
		timeTracker.Sleeping()
		time.Sleep(delay) // todo: make it cancelable
		timeTracker.Working()
		w.Status.DecSleeping()
	}

	w.Status.IncBusy()
	res := w.Nib.Punch(item)
	res.StartMissed = res.StartTime.Sub(expectedStart)
	res.ExtractValues(item.RegexOut, w.Values)
	res.ReqBytes = item.Payload
	res.Label = item.Label
	w.Output.Push(res)
	w.Status.DecBusy()
	w.Status.DecWorking()
	return false
}

func NewBasicWorker(name string, abort chan struct{}, wl *BaseWorkload, scheduleChan ScheduleChannel) *Worker {
	b := &Worker{
		Name:          name,
		Nib:           wl.NibMaker(),
		Abort:         abort,
		InputPayload:  wl.InputPayload(),
		InputSchedule: scheduleChan,
		Output:        wl.Output,
		StartTime:     wl.StartTime,
		Status:        wl.Status,
	}
	return b
}

type TimeTracker struct {
	Stamp         time.Time
	IsSleeping    bool
	SpentWorking  time.Duration
	SpentSleeping time.Duration
}

func (t *TimeTracker) Working() {
	t.SpentSleeping += time.Now().Sub(t.Stamp)
	t.Stamp = time.Now()
}

func (t *TimeTracker) Sleeping() {
	t.SpentWorking += time.Now().Sub(t.Stamp)
	t.Stamp = time.Now()
}

func (t *TimeTracker) Reset() {
	t.SpentWorking = 0
	t.SpentSleeping = 0
	t.Stamp = time.Now()
}

// something spawns workers on-demand - either on schedule, or to sustain hits/s
// each worker knows its input reader, nib (can be different nibs btw), output writer
// auto-USL scenario, scheduled scenario, external "stpd" scenario
// spit out some stats each N seconds

type WorkerSpawner interface {
	Run()
	GenerateSchedule() ScheduleChannel
	Interrupt()
}
