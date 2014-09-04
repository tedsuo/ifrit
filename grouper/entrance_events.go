package grouper

import (
	"sync"

	"github.com/tedsuo/ifrit"
)

type EntranceEvent struct {
	Member  Member
	Process ifrit.Process
}

type entranceEventChannel chan EntranceEvent

func newEntranceEventChannel(bufferSize int) entranceEventChannel {
	return make(entranceEventChannel, bufferSize)
}

type entranceEventBroadcaster struct {
	channels   []entranceEventChannel
	buffer     slidingBuffer
	bufferSize int
	lock       *sync.RWMutex
}

func newEntranceEventBroadcaster(bufferSize int) *entranceEventBroadcaster {
	return &entranceEventBroadcaster{
		channels:   make([]entranceEventChannel, 0),
		buffer:     newSlidingBuffer(bufferSize),
		bufferSize: bufferSize,
		lock:       new(sync.RWMutex),
	}
}

func (b *entranceEventBroadcaster) Attach() entranceEventChannel {
	b.lock.Lock()
	defer b.lock.Unlock()

	channel := newEntranceEventChannel(b.bufferSize)
	b.buffer.Range(func(event interface{}) {
		channel <- event.(EntranceEvent)
	})
	b.channels = append(b.channels, channel)
	return channel
}

func (b *entranceEventBroadcaster) Broadcast(entrance EntranceEvent) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	b.buffer.Append(entrance)

	for _, entranceChan := range b.channels {
		entranceChan <- entrance
	}
}

func (b *entranceEventBroadcaster) Close() {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, channel := range b.channels {
		close(channel)
	}
	b.channels = nil
}
