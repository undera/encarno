package core

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

type Status struct {
	sleeping int64
	busy     int64
	working  int64
	waiting  int64
	missed   int64
	cnt      int64
	mx       *sync.Mutex
}

func NewStatus() *Status {
	return &Status{
		mx: new(sync.Mutex),
	}
}

func (o *Status) IncWaiting() {
	atomic.AddInt64(&o.waiting, 1)
}

func (o *Status) DecWaiting() {
	atomic.AddInt64(&o.waiting, -1)
}

func (o *Status) GetWaiting() int64 {
	return o.waiting
}

func (o *Status) GetWorking() int64 {
	return o.working
}

func (o *Status) GetSleeping() int64 {
	return o.sleeping
}

func (o *Status) GetBusy() int64 {
	return o.busy
}

func (o *Status) IncWorking() {
	atomic.AddInt64(&o.working, 1)
}

func (o *Status) DecWorking() {
	atomic.AddInt64(&o.working, -1)
	if o.working < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *Status) IncSleeping() {
	atomic.AddInt64(&o.sleeping, 1)
	if o.sleeping < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *Status) DecSleeping() {
	atomic.AddInt64(&o.sleeping, -1)
}

func (o *Status) DecBusy() {
	atomic.AddInt64(&o.busy, -1)
	if o.busy < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *Status) IncBusy() {
	atomic.AddInt64(&o.busy, 1)
}

func (o *Status) Start() {
	go func() {
		for range time.NewTicker(1 * time.Second).C {
			working := o.GetWorking()
			sleeping := o.GetSleeping()
			busy := o.GetBusy()
			waiting := o.GetWaiting()

			o.mx.Lock()
			cnt := o.cnt
			o.cnt = 0
			missed := o.missed
			o.missed = 0
			o.mx.Unlock()

			// start to worry if we fail to serve load properly
			miss := time.Duration(0)
			if cnt > 0 {
				miss = time.Duration(missed / cnt).Round(10 * time.Millisecond)
			}

			log.Infof("Workers: waiting: %d, working: %d, sleeping: %d, busy: %d, lag: %s, reqs: %d", waiting, working, sleeping, busy, miss, cnt)
		}
	}()
}

func (o *Status) StartMissed(sub time.Duration) {
	o.mx.Lock()
	o.missed += sub.Nanoseconds()
	o.cnt++
	o.mx.Unlock()
}
