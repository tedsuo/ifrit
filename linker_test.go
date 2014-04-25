package ifrit_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"

	"github.com/tedsuo/ifrit"
)

var _ = Describe("Linker", func() {
	var pingProc1 ifrit.Process
	var pingProc2 ifrit.Process
	var errChan chan error

	BeforeEach(func() {
		pingProc1 = ifrit.Envoke(make(PingChan))
		pingProc2 = ifrit.Envoke(make(PingChan))
		errChan = make(chan error)
	})

	Describe("Link()", func() {
		var linkedProc ifrit.Process

		BeforeEach(func() {
			linkedProc = ifrit.Link(ifrit.Group{
				"ping1": pingProc1,
				"ping2": pingProc2,
			})
		})

		Context("when the linked process is signaled to stop", func() {
			BeforeEach(func() {
				linkedProc.Signal(os.Kill)
			})

			It("exits with a combined error message", func() {
				err := linkedProc.Wait()
				errMsg := err.Error()
				Ω(errMsg).Should(ContainSubstring("ping1"))
				Ω(errMsg).Should(ContainSubstring("ping2"))
				Ω(errMsg).Should(ContainSubstring(PingExitSignal.Error()))
			})

			It("propagates signals to the process group", func() {
				err1 := pingProc1.Wait()
				Ω(err1).Should(Equal(PingExitSignal))

				err2 := pingProc2.Wait()
				Ω(err2).Should(Equal(PingExitSignal))
			})
		})
	})
})
