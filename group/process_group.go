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
	count := len(rGroup)
	p := make(processGroup,count)
	mChan := make(MemberChan, count)

	for name, runner := range rGroup {
		go mChan.envokeMember(name, runner)
	}
	for i := 0; i < count; i++ {
		p[i] = <-mChan
	}
	return p
}

type Member struct {
	Name    string
	Process ifrit.Process
}

type MemberChan chan Member

func (mChan MemberChan) envokeMember(name string, runner ifrit.Runner) {
	mChan <- Member{Name: name, Process: ifrit.Envoke(runner)}
}

type processGroup []Member

func (group processGroup) Signal(signal os.Signal) {
	for _, m := range group {
		m.Process.Signal(signal)
	}
}

func (group processGroup) waitForGroup() error {
	var errMsg string
	for _, m := range group {
		err := <-m.Process.Wait()
		if err != nil {
			errMsg += fmt.Sprintf("%s: %s/n", m.Name, err)
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
