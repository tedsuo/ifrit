package ifrit_test

import (
	"errors"
	"os"
)

type Ping struct{}

var PingerExitedFromPing = errors.New("pinger exited with a ping")
var PingerExitedFromSignal = errors.New("pinger exited with a signal")

type PingChan chan Ping

func (p PingChan) Run(sigChan <-chan os.Signal) error {
	select {
	case <-sigChan:
		return PingerExitedFromSignal
	case p <- Ping{}:
		return PingerExitedFromPing
	}
}
