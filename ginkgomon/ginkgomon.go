package ginkgomon

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

type Runner struct {
	Name              string
	BinPath           string
	AnsiColorCode     string
	StartCheck        string
	StartCheckTimeout time.Duration
	Args              []string
}

func (r *Runner) Run(sigChan <-chan os.Signal, ready chan<- struct{}) error {
	session, err := gexec.Start(
		exec.Command(
			r.BinPath,
			r.Args...,
		),
		gexec.NewPrefixedWriter(fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name), ginkgo.GinkgoWriter),
		gexec.NewPrefixedWriter(fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name), ginkgo.GinkgoWriter),
	)

	Ω(err).ShouldNot(HaveOccurred())

	if r.StartCheck != "" {
		timeout := r.StartCheckTimeout
		if timeout == 0 {
			timeout = time.Second
		}

		Eventually(session, timeout).Should(gbytes.Say(r.StartCheck))
	}

	close(ready)

	var signal os.Signal

	for {
		select {

		case signal = <-sigChan:
			session.Signal(signal)

		case <-session.Exited:
			if session.ExitCode() == 0 {
				return nil
			}
			return fmt.Errorf("exit status %d", session.ExitCode())
		}
	}
}
