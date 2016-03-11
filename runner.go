package ifrit

import "os"

type Signal interface {
  os.Signal
  context.Context
}

type signal struct{
  os.Signal
  context.Context
}

func NewSignal(sig os.Signal,ctx context.Context) Signal {
  return &signal{sig,ctx}
}

type Context interface {
  Ready()
  Signals() <-chan Signal
  Span() opentracing.Span
}

type context struct{
  signalChan
  readyChan
  opentracing.Span
}

func NewContext(s opentracing.Span) Context {
  return context{
  make(signalChan),
  make(readyChan),
  s,
  }
}

type signalChan chan Signal

func (s signalChan) Signals() <-chan Signal {
  return s
}

type readyChan chan struct{}

func (r readyChan) Ready() {
  close(r)
}

/*
A Runner defines the contents of a Process. A Runner implementation performs an
aribtrary unit of work, while waiting for a shutdown signal. The unit of work
should avoid any orchestration. Instead, it should be broken down into simpler
units of work in seperate Runners, which are then orcestrated by the ifrit
standard library.

An implementation of Runner has the following responibilities:

 - setup within a finite amount of time.
 - close the ready channel when setup is complete.
 - once ready, perform the unit of work, which may be infinite.
 - respond to shutdown signals by exiting within a finite amount of time.
 - return nil if shutdown is successful.
 - return an error if an exception has prevented a clean shutdown.

By default, Runners are not considered restartable; Run will only be called once.
See the ifrit/restart package for details on restartable Runners.
*/
type Runner interface {
	Run(Context) error
}

/*
The RunFunc type is an adapter to allow the use of ordinary functions as Runners.
If f is a function that matches the Run method signature, RunFunc(f) is a Runner
object that calls f.
*/
type RunFunc func(Context) error

func (r RunFunc) Run(ctx Context) error {
	return r(ctx)
}
