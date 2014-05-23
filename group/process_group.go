package group

import (
	"errors"
	"fmt"
	"os"
	"github.com/tedsuo/ifrit"
)

type ProcessGroup interface{
	ifrit.Process
	//Exit() MemberChan
}

func EnvokeGroup(rGroup RunGroup) ProcessGroup {
	p := processGroup{}
	count := len(rGroup)
	mChan := make(MemberChan, count)

	for name, runner := range rGroup {
		go mChan.envokeMember(name, runner)
	}
	for i := 0; i < count; i++ {
		m := <-mChan
		p[m.name] = m.process
	}
	return p
}

type Member struct {
	name    string
	process ifrit.Process
}

type MemberChan chan Member

func (mChan MemberChan) envokeMember(name string, runner ifrit.Runner) {
	mChan <- Member{name: name, process: ifrit.Envoke(runner)}
}

type processGroup map[string]ifrit.Process

func (group processGroup) Signal(signal os.Signal) {
	for _, p := range group {
		p.Signal(signal)
	}
}

func (group processGroup) waitForGroup() error {
	var errMsg string
	for name, p := range group {
		err := <-p.Wait()
		if err != nil {
			errMsg += fmt.Sprintf("%s: %s/n", name, err)
		}
	}

	var err error
	if errMsg != "" {
		err = errors.New(errMsg)
	}
	return err
}

func (group processGroup) Wait() <-chan error {
	errChan := make(chan error, 1)

	go func() {
		errChan <- group.waitForGroup()
	}()

	return errChan
}
