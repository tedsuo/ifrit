package ifrit

import (
	"errors"
	"fmt"
	"os"
)

type Group map[string]Process

func Link(pGroup Group) Process {
	return Envoke(newLinkedRunner(pGroup))
}

type linkedProcess struct {
	pGroup   Group
	exitChan chan error
}

func newLinkedRunner(pGroup Group) Runner {
	p := &linkedProcess{
		pGroup:   pGroup,
		exitChan: make(chan error),
	}
	go p.waitForExit()
	return p
}

func (p *linkedProcess) Run(sig <-chan os.Signal) error {
	for {
		select {
		case signal := <-sig:
			p.broadcastSignal(signal)
		case err := <-p.exitChan:
			return err
		}
	}
	return nil
}

func (p *linkedProcess) broadcastSignal(signal os.Signal) {
	for _, proc := range p.pGroup {
		proc.Signal(signal)
	}
}

func (p *linkedProcess) waitForExit() {
	var errMsg string
	for name, proc := range p.pGroup {
		err := proc.Wait()
		if err != nil {
			errMsg += fmt.Sprintf("%s: %s/n", name, err)
		}
	}

	if errMsg != "" {
		p.exitChan <- errors.New(errMsg)
	}
	p.exitChan <- nil
}
