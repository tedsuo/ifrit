package ifrit

import "os"

/*
A Process represents a Runner that has been started.  It is safe to call any
method on a Process even after the Process has exited.
*/
type Process interface {
	// Ready returns a channel which will close once the runner is active
	Ready() <-chan struct{}

	// Wait returns a channel that will emit a single error once the Process exits.
	Wait() <-chan error

	// Signal sends a shutdown signal to the Process.  It does not block.
	Signal(Signal)
}

/*
Invoke executes a Runner and returns a Process once the Runner is ready.  Waiting
for ready allows program initializtion to be scripted in a procedural manner.
To orcestrate the startup and monitoring of multiple Processes, please refer to
the ifrit/grouper package.
*/
func Invoke(span opentracing.Span, r Runner) Process {
	p := Background(span,r)

	select {
	case <-p.Ready():
	case <-p.Wait():
	}

	return p
}

/*
Background executes a Runner and returns a Process immediately, without waiting.
*/
func Background(span opentracing.Span, r Runner) Process {
	p := newProcess(span,r)
	go p.run()
	return p
}

type process struct {
	ctx context
	runner     Runner
	exited     chan struct{}
	exitStatus error
}

func newProcess(span opentracing.Span,runner Runner) *process {
	return &process{
		ctx: NewContext(span),
		runner:  runner,
		exited:  make(chan struct{}),
	}
}

func (p *process) run() {
	p.exitStatus = p.runner.Run(p.ctx)
	close(p.exited)
}

func (p *process) Ready() <-chan struct{} {
	return p.ctx.readyChan
}

func (p *process) Wait() <-chan error {
	exitChan := make(chan error, 1)

	go func() {
		<-p.exited
		exitChan <- p.exitStatus
	}()

	return exitChan
}

func (p *process) Signal(signal Signal) {
	go func() {
		select {
		case p.ctx.signalChan <- signal:
		case <-p.exited:
		}
	}()
}
