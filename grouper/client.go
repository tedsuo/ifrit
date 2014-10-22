package grouper

import (
	"sync"

	"github.com/tedsuo/ifrit"
)

/*
StaticClient provides a client with event notifications, via channels.
Event channels are independent of each other. Each event channel has a buffer,
which records events and will drop events once the buffer is filled.
*/
type StaticClient interface {

/*
EntranceListener provides a new buffered channel of entrance events, which are
emited every time an inserted process is ready. To help prevent race conditions,
every new channel is populated with previously emited events, up to it's buffer
size.
*/
	EntranceListener() <-chan EntranceEvent

/*
ExitListener provides a new buffered channel of exit events, which are emited
every time an inserted process is ready. To help prevent race conditions, every
new channel is populated with previously emited events, up to it's buffer size.
*/
	ExitListener() <-chan ExitEvent

/*
CloseNotifier provides a new unbuffered channel, which will emit a single event
once the group has been closed.
*/
	CloseNotifier() <-chan struct{}
}

/*
DynamicClient provides a client with group controls and event notifications.
A client can use the insert channel to add members to the group. When the group
becomes full, the insert channel blocks until a running process exits the group.
Once there are no more members have been added, the client can close the dynamic
group, which causes it to become a static group.
*/
type DynamicClient interface {

/*
Inserter provides an unbuffered channel for adding members to a group. When the
group becomes full, the insert channel blocks until a running process exits.
Once the group is closed, insert channels block forever.
*/
	Inserter() chan<- Member

/*
Close causes a dynamic group to become a static group. This means that no new
members may be inserted, and the group will exit once all members have
completed.
*/
	Close()

/*
See the StaticClient interface for documentation on event listeners.
*/
	StaticClient
}

/*
poolClient implements DynamicClient.
*/
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

func (c poolClient) Inserter() chan<- Member {
	return c.insertChannel
}

func (c poolClient) insertEventListener() <-chan Member {
	return c.insertChannel
}

func (c poolClient) EntranceListener() <-chan EntranceEvent {
	return c.entranceBroadcaster.Attach()
}

func (c poolClient) broadcastEntrance(event EntranceEvent) {
	c.entranceBroadcaster.Broadcast(event)
}

func (c poolClient) closeEntranceBroadcaster() {
	c.entranceBroadcaster.Close()
}

func (c poolClient) ExitListener() <-chan ExitEvent {
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
