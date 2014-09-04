package grouper

import (
	"sync"

	"github.com/tedsuo/ifrit"
)

type StaticClient interface {
	NewEntranceListener() <-chan EntranceEvent
	NewExitListener() <-chan ExitEvent
}

type PoolClient interface {
	NewEntranceListener() <-chan EntranceEvent
	NewExitListener() <-chan ExitEvent
	Insert() chan<- Member
	CloseNotifier() <-chan struct{}
	Close()
}

type poolClient struct {
	insertChannel       chan Member
	closeNotifier       chan struct{}
	closeOnce           *sync.Once
	entranceBroadcaster *entranceEventBroadcaster
	exitBroadcaster     *exitEventBroadcaster
}

func newPoolClient(bufferSize int) poolClient {
	return poolClient{
		insertChannel:       make(chan Member),
		closeNotifier:       make(chan struct{}),
		closeOnce:           new(sync.Once),
		entranceBroadcaster: newEntranceEventBroadcaster(bufferSize),
		exitBroadcaster:     newExitEventBroadcaster(bufferSize),
	}
}

func (c poolClient) Get(Member) (ifrit.Process, bool) {
	return nil, false
}

func (c poolClient) Insert() chan<- Member {
	return c.insertChannel
}

func (c poolClient) insertEventListener() <-chan Member {
	return c.insertChannel
}

func (c poolClient) NewEntranceListener() <-chan EntranceEvent {
	return c.entranceBroadcaster.Attach()
}

func (c poolClient) broadcastEntrance(event EntranceEvent) {
	c.entranceBroadcaster.Broadcast(event)
}

func (c poolClient) closeEntranceBroadcaster() {
	c.entranceBroadcaster.Close()
}

func (c poolClient) NewExitListener() <-chan ExitEvent {
	return c.exitBroadcaster.Attach()
}

func (c poolClient) broadcastExit(event ExitEvent) {
	c.exitBroadcaster.Broadcast(event)
}

func (c poolClient) closeExitBroadcaster() {
	c.exitBroadcaster.Close()
}

func (c poolClient) closeBroadcasters() error {
	c.entranceBroadcaster.Close()
	c.exitBroadcaster.Close()
	return nil
}

func (c poolClient) Close() {
	c.closeOnce.Do(func() {
		close(c.closeNotifier)
	})
}

func (c poolClient) CloseNotifier() <-chan struct{} {
	return c.closeNotifier
}
