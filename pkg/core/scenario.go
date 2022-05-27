package core

// something spawns workers on-demand - either on schedule, or to sustain hits/s
// each worker knows its input reader, nib (can be different nibs btw), output writer
// auto-USL scenario, scheduled scenario, external "stpd" scenario
// spit out some stats each N seconds

type Scenario interface {
	Run()
}
