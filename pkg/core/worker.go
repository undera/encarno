package core

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// basic worker and regex-capable worker, regex-capable should read file on its own
// track expected request time and factual, report own overloaded state, auto-stop if unable to conform

type Worker struct {
	Index          int
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
	stopped        bool
}

func (w *Worker) Run() {
outer:
	for !w.stopped {
		// TODO: only measure single iteration with time tracker, record its ratio into result
		select {
		case <-w.Abort:
			log.Debugf("Aborting worker: %s", w.Index)
			break outer
		default:
			shouldStop := w.Iteration()
			if shouldStop {
				break outer
			}
		}
	}
	log.Infof("Worker finished: %s", w.Index)
	w.Finished = true
	// TODO: somehow notify workers array/count
}

func (w *Worker) Iteration() bool {
	w.Status.IncWaiting()
	offset := <-w.InputSchedule
	w.Status.DecWaiting()

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
		log.Debugf("[%s] Sleeping: %dns", w.Index, delay)
		w.Status.IncSleeping()
		time.Sleep(delay) // todo: make it cancelable
		w.Status.DecSleeping()

	}

	if !w.stopped {
		w.Status.IncBusy()
		res := w.Nib.Punch(item)
		res.StartTS = res.StartTime.Unix() // TODO: use nanoseconds
		res.Worker = w.Index
		if res.Error != nil {
			res.ErrorStr = res.Error.Error()
		}
		//res.StartMissed = res.StartTime.Sub(expectedStart)
		res.ExtractValues(item.RegexOut, w.Values)
		res.ReqBytes = item.Payload
		if item.Label != "" { // allow Nib to generate own label
			res.Label = item.Label
		}
		res.Concurrency = w.Status.GetBusy()
		w.Output.Push(res)
		w.Status.DecBusy()
	}
	w.Status.DecWorking()
	return false
}

func (w *Worker) Stop() {
	w.stopped = true
}

func NewBasicWorker(index int, abort chan struct{}, wl *BaseWorkload, scheduleChan ScheduleChannel) *Worker {
	b := &Worker{
		Index:         index,
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

type WorkerSpawner interface {
	Run()
	GenerateSchedule() ScheduleChannel
	Interrupt()
}
