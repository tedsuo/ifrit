package grouper

import "github.com/tedsuo/ifrit"

type Member struct {
	Name string
	ifrit.Runner
}

type Members []Member

type pMember struct {
	Member
	ifrit.Process
}
