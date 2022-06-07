package core

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// basic worker and regex-capable worker, regex-capable should read file on its own
// track expected request time and factual, report own overloaded state, auto-stop if unable to conform

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
	Status         *Status
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
	w.Status.IncWaiting()
	offset := <-w.InputSchedule
	w.Status.DecWaiting()
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
	res.StartTS = res.StartTime.Unix()
	if res.Error != nil {
		res.ErrorStr = res.Error.Error()
	}
	res.StartMissed = res.StartTime.Sub(expectedStart)
	res.ExtractValues(item.RegexOut, w.Values)
	res.ReqBytes = item.Payload
	if item.Label != "" { // allow Nib to generate own label
		res.Label = item.Label
	}
	res.Concurrency = w.Status.GetBusy()
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
