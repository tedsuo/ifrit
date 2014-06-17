package grouper_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper/0.2"
	"github.com/tedsuo/ifrit/test_helpers"
)

var _ = Describe("Members", func() {
	Describe("ifrit.Envoke()", func() {
		var pinger1 test_helpers.PingChan
		var pinger2 test_helpers.PingChan
		var pGroup ifrit.Process

		BeforeEach(func() {
			pinger1 = make(test_helpers.PingChan)
			pinger2 = make(test_helpers.PingChan)
			pGroup = ifrit.Envoke(grouper.Members{
				{"ping1", pinger1, grouper.RestartMePolicy()},
				{"ping2", pinger2, grouper.StopGroupPolicy()},
			})
		})

		Context("when the linked process is signaled to stop", func() {
			BeforeEach(func() {
				pGroup.Signal(os.Kill)
			})

			It("exits with nil", func() {
				Eventually(pGroup.Wait()).Should(Receive(nil))
			})
		})
	})
})
