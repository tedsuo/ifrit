package group_test

import (
	"os"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/group"
	"github.com/tedsuo/ifrit/test_helpers"
)

var _ = Describe("ProcessGroup", func() {
	var rGroup group.RunGroup

	BeforeEach(func() {
		rGroup = group.RunGroup{
			"ping1": make(test_helpers.PingChan),
			"ping2": make(test_helpers.PingChan),
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
				Ω(errMsg).Should(ContainSubstring(test_helpers.PingerExitedFromSignal.Error()))
			})
		})
	})
})
