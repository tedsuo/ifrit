package ifrit

import (
	"os"
	"sync"
)

type Runner interface {
	Run(<-chan os.Signal) error
}

type RunFunc func(<-chan os.Signal) error

func (r RunFunc) Run(sig <-chan os.Signal) error {
	return r(sig)
}

type Process interface {
	Wait() error
	Signal(os.Signal)
}

func Envoke(r Runner) Process {
	p := &process{
		runner:         r,
		sig:            make(chan os.Signal),
		exitStatusChan: make(chan error, 1),
	}
	go p.run()
	return p
}

type process struct {
	runner         Runner
	sig            chan os.Signal
	exitStatus     error
	exitStatusChan chan error
	exitOnce       sync.Once
}

func (p *process) run() {
	p.exitStatusChan <- p.runner.Run(p.sig)
}

func (p *process) Wait() error {
	p.exitOnce.Do(func() {
		p.exitStatus = <-p.exitStatusChan
	})
	return p.exitStatus
}

func (p *process) Signal(signal os.Signal) {
	go func() {
		p.sig <- signal
	}()
}
