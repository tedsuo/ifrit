package grouper

import (
	"os"

	"github.com/tedsuo/ifrit"
)

type Name string

type Members []Member

type Member struct {
	Name
	ifrit.Loader
	RestartPolicy
}

func (r Members) Load() (ifrit.Runner, bool) {
	return r, true
}

func (r Members) Run(sig <-chan os.Signal, ready chan<- struct{}) error {
	signaledToStop := false

	group := make(pGroup, len(r))

	pmChan := make(pMemberChan, len(r))
	pmChan.envokeGroup(r, group)

	exitChan := make(exitChannel, len(group))
	exitChan.waitForGroup(group)

	desiredCount := len(group)

	if ready != nil {
		close(ready)
	}

	for {
		if desiredCount == 0 {
			return nil
		}

		select {

		case signal := <-sig:
			signaledToStop = true
			group.Signal(signal)

		case pm := <-pmChan:
			group[pm.Process] = pm
			exitChan.waitForProcess(pm.Process)

		case e := <-exitChan:
			if signaledToStop {
				delete(group, e.Process)
				desiredCount--
				continue
			}

			m := group[e.Process]
			restart := m.RestartPolicy()

			if restart.Signal != Continue {
				group.Signal(restart.Signal)
			}

			if !restart.AttemptRestart {
				if restart.Signal != Continue {
					signaledToStop = true
				}
				delete(group, e.Process)
				desiredCount--
				continue
			}

			lr, ok := m.Runner.(ifrit.LoadRunner)
			if !ok {
				delete(group, e.Process)
				desiredCount--
				continue
			}

			r, ok := lr.Load()
			if !ok {
				delete(group, e.Process)
				desiredCount--
				continue
			}

			desiredCount++
			go pmChan.envokeMember(Member{m.Name, r, m.RestartPolicy})
		}
	}
}

type exit struct {
	ifrit.Process
	error
}

type exitChannel chan exit

func (exitChan exitChannel) waitForGroup(group pGroup) {
	for p, _ := range group {
		exitChan.waitForProcess(p)
	}
}

func (exitChan exitChannel) waitForProcess(p ifrit.Process) {
	go func() {
		err := <-p.Wait()
		exitChan <- exit{p, err}
	}()
}
