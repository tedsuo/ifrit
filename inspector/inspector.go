package inspector

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

const SIGNAL_BUFFER_SIZE = 1024

type Inspector struct {
	StackPath string
	Signals   []os.Signal
}

func New(stackPath string, signals ...os.Signal) Inspector {
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGUSR2}
	}
	return Inspector{
		StackPath: stackPath,
		Signals:   signals,
	}
}

func (i Inspector) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	osSignals := make(chan os.Signal, SIGNAL_BUFFER_SIZE)
	signal.Notify(osSignals, i.Signals...)

dance:
	for {
		select {
		case sig := <-signals:
			for writeSignal := range i.Signals {
				if sig == writeSignal {
					i.writeStackTrace()
					break dance
				}
			}
			return nil

		case <-osSignals:
			i.writeStackTrace()
		}
	}

	return nil
}

func (i Inspector) writeStackTrace() {
	f, err := os.Create(i.StackPath)
	if err != nil {
		log.Println("Inspector failed to create stack file", i.StackPath, err)
	}

	by := make([]byte, 1024*1024*100)
	n := runtime.Stack(by, true)

	_, err = f.Write(by[:n])
	if err != nil {
		log.Println("Inspector failed to write to stack file", i.StackPath, err)
	}

	log.Println("Inspector wrote stack to", i.StackPath)
}
