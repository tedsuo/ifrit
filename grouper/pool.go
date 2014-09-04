package grouper

import (
	"fmt"
	"os"

	"github.com/tedsuo/ifrit"
)

type Pool struct {
	client   poolClient
	signal   os.Signal
	poolSize int
}

func NewPool(signal os.Signal, poolSize int, eventBufferSize int) *Pool {
	return &Pool{
		client:   newPoolClient(eventBufferSize),
		poolSize: poolSize,
		signal:   signal,
	}
}

func (p *Pool) Client() PoolClient {
	return p.client
}

func (p *Pool) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	processes := newProcessSet()
	insertEvents := p.client.insertEventListener()
	closeNotifier := p.client.CloseNotifier()
	entranceEvents := make(entranceEventChannel)
	exitEvents := make(exitEventChannel)

	invoking := 0
	close(ready)

	for {
		select {
		case shutdown := <-signals:
			processes.Signal(shutdown)
			p.client.Close()

		case <-closeNotifier:
			closeNotifier = nil
			insertEvents = nil
			if processes.Length() == 0 {
				return p.client.closeBroadcasters()
			}

		case newMember, ok := <-insertEvents:
			if !ok {
				p.client.Close()
				insertEvents = nil
				break
			}

			process := ifrit.Background(newMember)
			processes.Add(newMember, process)
			if processes.Length() == p.poolSize {
				insertEvents = nil
			}
			invoking++
			waitForEvents(newMember, process, entranceEvents, exitEvents)

		case entranceEvent := <-entranceEvents:
			invoking--
			p.client.broadcastEntrance(entranceEvent)

			if closeNotifier == nil && invoking == 0 {
				p.client.closeEntranceBroadcaster()
				entranceEvents = nil
			}

		case exitEvent := <-exitEvents:
			processes.Remove(exitEvent.Member)
			p.client.broadcastExit(exitEvent)

			if !processes.Signaled() && p.signal != nil {
				processes.Signal(p.signal)
				p.client.Close()
				insertEvents = nil
			}

			if processes.Complete() || (processes.Length() == 0 && insertEvents == nil) {
				return p.client.closeBroadcasters()
			}

			if !processes.Signaled() {
				insertEvents = p.client.insertEventListener()
			}
		}
	}
}

func waitForEvents(member Member, process ifrit.Process, entrance entranceEventChannel, exit exitEventChannel) {
	go func() {
		select {
		case <-process.Ready():
			entrance <- EntranceEvent{
				Member:  member,
				Process: process,
			}

			exit <- ExitEvent{
				Member: member,
				Err:    <-process.Wait(),
			}

		case err := <-process.Wait():
			entrance <- EntranceEvent{
				Member:  member,
				Process: process,
			}

			exit <- ExitEvent{
				Member: member,
				Err:    err,
			}
		}
	}()
}

type processSet struct {
	processes map[Member]ifrit.Process
	shutdown  os.Signal
}

func newProcessSet() *processSet {
	return &processSet{
		processes: map[Member]ifrit.Process{},
	}
}

func (g *processSet) Signaled() bool {
	return g.shutdown != nil
}

func (g *processSet) Signal(signal os.Signal) {
	g.shutdown = signal
	for _, p := range g.processes {
		p.Signal(signal)
	}
}

func (g *processSet) Length() int {
	return len(g.processes)
}

func (g *processSet) Complete() bool {
	return len(g.processes) == 0 && g.shutdown != nil
}

func (g *processSet) Add(member Member, process ifrit.Process) {
	_, ok := g.processes[member]
	if ok {
		panic(fmt.Errorf("member inserted twice: %#v", member))
	}
	g.processes[member] = process
}

func (g *processSet) Remove(member Member) {
	delete(g.processes, member)
}
