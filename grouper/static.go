package grouper

import (
	"fmt"
	"os"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/proxy"
)

type StaticGroup interface {
	ifrit.Runner
	Client() StaticClient
}

func NewParallel(signal os.Signal, members []Member) StaticGroup {
	return newStatic(signal, members, parallelInit)
}

func parallelInit(members Members, client PoolClient) {
	insert := client.Insert()
	closed := client.CloseNotifier()

	for _, member := range members {
		select {
		case insert <- member:
		case <-closed:
			return
		}
	}
}

func NewOrdered(signal os.Signal, members []Member) StaticGroup {
	return newStatic(signal, members, orderedInit)
}

func orderedInit(members Members, client PoolClient) {
	entranceEvents := client.NewEntranceListener()
	insert := client.Insert()
	closed := client.CloseNotifier()

	for _, member := range members {
		select {
		case insert <- member:
		case <-closed:
			return
		}
		<-entranceEvents
	}
}

func NewSerial(members []Member) StaticGroup {
	return newStatic(nil, members, serialInit)
}

func serialInit(members Members, client PoolClient) {
	exitEvents := client.NewExitListener()
	insert := client.Insert()
	closed := client.CloseNotifier()

	for _, member := range members {
		select {
		case insert <- member:
		case <-closed:
			return
		}

		exit := <-exitEvents
		if exit.Err != nil {
			break
		}
	}
}

type staticGroup struct {
	pool *Pool
	Members
	Init func(members Members, client PoolClient)
}

func newStatic(signal os.Signal, members []Member, init func(members Members, client PoolClient)) StaticGroup {
	return staticGroup{
		pool:    NewPool(signal, len(members), len(members)),
		Members: members,
		Init:    init,
	}
}

func (g staticGroup) Client() StaticClient {
	return g.pool.Client()
}

func (g staticGroup) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	bufferSize := len(g.Members)

	client := g.pool.Client()

	go ifrit.Envoke(proxy.New(signals, g.pool))

	go func() {
		g.Init(g.Members, client)
		client.Close()
		close(ready)
	}()

	errorTrace := NewErrorTrace(bufferSize)
	errorTrace = errorTrace.TraceExitEvents(client.NewExitListener())
	return errorTrace.ToError()
}

type ErrorTrace []ExitEvent

func NewErrorTrace(bufferSize int) ErrorTrace {
	return make(ErrorTrace, 0, bufferSize)
}

func (trace ErrorTrace) TraceExitEvents(exitEvents <-chan ExitEvent) ErrorTrace {
	for exitEvent := range exitEvents {
		trace = append(trace, exitEvent)
	}
	return trace
}

func (trace ErrorTrace) ToError() error {
	for _, exit := range trace {
		if exit.Err != nil {
			return trace
		}
	}
	return nil
}

func (trace ErrorTrace) Error() string {
	msg := "Exit trace for group:\n"

	for _, exit := range trace {
		if exit.Err == nil {
			msg += fmt.Sprintf("%s exited with nil\n", exit.Member.Name)
		} else {
			msg += fmt.Sprintf("%s exited with error: %s\n", exit.Member.Name, exit.Err.Error())
		}
	}

	return msg
}
