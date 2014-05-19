package ifrit_test

import (
	"os"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tedsuo/ifrit"
)

var _ = Describe("ProcessGroup", func() {
	var rGroup ifrit.RunGroup

	BeforeEach(func() {
		rGroup = ifrit.RunGroup{
			"ping1": make(PingChan),
			"ping2": make(PingChan),
		}
	})

	Describe("Envoke()", func() {
		var pGroup ifrit.Process

		BeforeEach(func() {
			pGroup = ifrit.Envoke(rGroup)
		})

		Context("when the linked process is signaled to stop", func() {
			BeforeEach(func() {
				pGroup.Signal(os.Kill)
			})

			It("exits with a combined error message", func() {
				err := <-pGroup.Wait()
				errMsg := err.Error()
				Ω(errMsg).Should(ContainSubstring("ping1"))
				Ω(errMsg).Should(ContainSubstring("ping2"))
				Ω(errMsg).Should(ContainSubstring(PingerExitedFromSignal.Error()))
			})
		})
	})
})
