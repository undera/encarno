package core

import log "github.com/sirupsen/logrus"

// config
// input
// output
// workers pool
//	 spawner
//     nib maker

type Controller struct {
}

func (c *Controller) Interrupt() {
	panic("TODO") //todo tell workers that they're exiting after sleep ends
}

func (c *Controller) RunWithConfig(config Configuration, spawner WorkerSpawner) {
	var input Input
	iChan := input.Start(config.Input)

	var output Output
	output.Start(config.Output)

	c.Run(iChan, spawner)
}

func (c *Controller) Run(input InputChannel, spawner WorkerSpawner) {
	log.Debugf("Starting scenario 'external'")

	inputs := make(InputChannel)

	spawner.SpawnInitial(inputs)

	for x := range input {
		select {
		case inputs <- x: // try putting if somebody is reading it
			continue
		default:
			spawner.SpawnOnDemand(inputs, x)
			inputs <- x
		}

		if spawner.ShouldStop() {
			break
		}
	}
}
