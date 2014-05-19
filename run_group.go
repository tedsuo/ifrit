package ifrit

import "os"

type RunGroup map[string]Runner

func (r RunGroup) Run(sig <-chan os.Signal, ready chan<- struct{}) error {
	p := envokeGroup(r)

	if ready != nil {
		close(ready)
	}

	for {
		select {
		case signal := <-sig:
			p.Signal(signal)
		case err := <-p.Wait():
			return err
		}
	}
}
