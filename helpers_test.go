package ifrit_test

import (
	"errors"
	"os"
)

type Ping struct{}

var PingExitNormal = errors.New("ping exited normal")
var PingExitSignal = errors.New("ping exited with a signal")

type PingChan chan Ping

func (p PingChan) Run(sigChan <-chan os.Signal) error {
	select {
	case <-sigChan:
		return PingExitSignal
	case p <- Ping{}:
		return PingExitNormal
	}
}
