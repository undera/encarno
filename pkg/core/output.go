package core

type Output interface {
	DecBusy()
	IncBusy()
	Push(res *OutputItem)
	IncSleeping()
	DecSleeping()
	IncWorking()
	DecWorking()
	GetWorking() int64
	GetSleeping() int64
	GetBusy() int64
}

// get result from worker via channel
// write small binary results
// write full request/response for debugging
// write only failures request/response

type OutputItem struct {
	Start int64
	End   int64
}
