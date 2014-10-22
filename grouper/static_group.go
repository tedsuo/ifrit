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

type staticGroup struct {
	pool DynamicGroup
	Members
	Init func(members Members, client DynamicClient)
}

/*
NewStatic creates a static group which starts according to it's init function.
Within the init function, the static group acts as a dynamic group. Once the
init function returns, the group is closed and acts as static group.  Use this
lower-level constructor if the NewParallel, NewOrdered, or NewSerial strategies
are insufficient.
*/
func NewStatic(signal os.Signal, members []Member, init func(members Members, client DynamicClient)) StaticGroup {
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
	err := g.Validate()
	if err != nil {
		return err
	}

	bufferSize := len(g.Members)
	client := g.pool.Client()

	go ifrit.Envoke(proxy.New(signals, g.pool))

	go func() {
		g.Init(g.Members, client)
		client.Close()
		close(ready)
	}()

	return traceExitEvents(make(ErrorTrace, 0, bufferSize), client.ExitListener())
}

/*
ErrorTrace is an error returned by a static group if any member exited with an error.
*/
type ErrorTrace []ExitEvent

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

func traceExitEvents(trace ErrorTrace, exitEvents <-chan ExitEvent) error {
	errorOccurred := false
	for exitEvent := range exitEvents {
		if exitEvent.Err != nil {
			errorOccurred = true
		}
		trace = append(trace, exitEvent)
	}
	if errorOccurred {
		return trace
	} else {
		return nil
	}
}
