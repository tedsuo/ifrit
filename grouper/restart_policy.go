package grouper

import (
	"os"
	"syscall"
	"time"
)

const Continue = Signal("continue")

type Signal string

func (s Signal) Signal() {}

func (s Signal) String() string {
	return string(s)
}

const (
	NoTimeout           = time.Duration(0)
	DefaultStopTimeout  = 10 * time.Second
	DefaultGraceTimeout = 5 * time.Minute
)

type Restart struct {
	AttemptRestart bool
	os.Signal
	time.Duration
}

type RestartPolicy func() Restart

var (
	RestartMePolicy = RestartPolicy(func() Restart {
		return Restart{true, Continue, NoTimeout}
	})

	StopMePolicy = RestartPolicy(func() Restart {
		return Restart{false, Continue, NoTimeout}
	})

	RestartGroupPolicy = RestartPolicy(func() Restart {
		return Restart{true, syscall.SIGINT, DefaultGraceTimeout}
	})

	StopGroupPolicy = RestartPolicy(func() Restart {
		return Restart{false, syscall.SIGTERM, DefaultGraceTimeout}
	})
)
