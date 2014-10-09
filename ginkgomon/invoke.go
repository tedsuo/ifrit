package ginkgomon

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/tedsuo/ifrit"
)

func Invoke(runner ifrit.Runner) ifrit.Process {
	process := ifrit.Background(runner)

	select {
	case <-process.Ready():
	case err := <-process.Wait():
		ginkgo.Fail(fmt.Sprintf("process failed to start: %s", err))
	}

	return process
}
