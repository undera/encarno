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
	Output         *Output
	Values         map[string][]byte
	Finished       bool
	IterationCount int
	Status         *Status

	stopped bool
}

func (w *Worker) Run() {
outer:
	for !w.stopped {
		// TODO: only measure single iteration with time tracker, record its ratio into result
		select {
		case <-w.Abort:
			log.Debugf("Aborting worker: %d", w.Index)
			break outer
		default:
			shouldStop := w.Iteration()
			if shouldStop {
				break outer
			}
		}
	}
	log.Infof("Worker finished: %d", w.Index)
	w.Finished = true
	// TODO: somehow notify workers array/count
}

func (w *Worker) Iteration() bool {
	w.Status.IncWaiting()
	offset := <-w.InputSchedule
	item := <-w.InputPayload
	if item == nil {
		return true
	}
	item.ReplaceValues(w.Values)
	w.Status.DecWaiting()

	w.Status.IncWorking()
	w.IterationCount += 1

	expectedStart := w.StartTime.Add(offset)
	delay := expectedStart.Sub(time.Now())
	if delay > 0 {
		log.Debugf("[%d] Sleeping: %dns", w.Index, delay)
		w.Status.IncSleeping()
		time.Sleep(delay) // todo: make it cancelable
		w.Status.DecSleeping()

	}

	if !w.stopped {
		item.ResolveStrings()
		res := w.DoBusy(item)
		w.Status.StartMissed(res.StartTime.Sub(expectedStart))
	}
	w.Status.DecWorking()
	return false
}

func (w *Worker) DoBusy(item *PayloadItem) *OutputItem {
	w.Status.IncBusy()
	res := w.Nib.Punch(item)
	res.StartTS = uint32(res.StartTime.Unix()) // TODO: use nanoseconds
	res.Worker = uint32(w.Index)
	res.ReqBytes = item.Payload

	if res.Label == "" { // allow Nib to generate own label
		res.Label = item.Label
	}

	if res.LabelIdx == 0 { // allow Nib to generate own label index
		res.LabelIdx = item.LabelIdx
	}

	res.Concurrency = uint32(w.Status.GetBusy())
	w.Status.DecBusy()
	res.ExtractValues(item.RegexOut, w.Values)
	res.Assert(item.Asserts)
	w.Output.Push(res)
	return res
}

func (w *Worker) Stop() {
	w.stopped = true
}

func NewBasicWorker(index int, abort chan struct{}, wl *BaseWorkload, scheduleChan ScheduleChannel, values ValMap) *Worker {
	// each worker gets own copy of values
	// TODO: should each input iteration reset those values?
	valuesCopy := make(ValMap)
	for k, v := range values {
		valuesCopy[k] = v
	}

	b := &Worker{
		Index:         index,
		Nib:           wl.NibMaker(),
		Abort:         abort,
		InputPayload:  wl.InputPayload(),
		InputSchedule: scheduleChan,
		Output:        wl.Output,
		StartTime:     wl.StartTime,
		Status:        wl.Status,
		Values:        valuesCopy,
	}
	return b
}

type WorkerSpawner interface {
	Run()
	GenerateSchedule() ScheduleChannel
	Interrupt()
}
