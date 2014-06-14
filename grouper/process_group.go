package grouper

import (
	"os"

	"github.com/tedsuo/ifrit"
)

type pGroup map[ifrit.Process]pMember

func (group pGroup) Signal(signal os.Signal) {
	for p, _ := range group {
		p.Signal(signal)
	}
}

type pMember struct {
	Member
	ifrit.Process
}

type pMemberChan chan pMember

func (pmChan pMemberChan) envokeMember(member Member) {
	pmChan <- pMember{Member: member, Process: ifrit.Envoke(member)}
}

func (pmChan pMemberChan) envokeGroup(group Members, p pGroup) {
	for _, member := range group {
		go pmChan.envokeMember(member)
	}

	for _ = range group {
		pm := <-pmChan
		p[pm.Process] = pm
	}
}
