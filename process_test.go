package ifrit_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"os"
)

var _ = Describe("Process", func() {
	Context("when a process is envoked", func() {
		var pinger PingChan
		var pingProc ifrit.Process
		var errChan chan error

		BeforeEach(func() {
			pinger = make(PingChan)
			pingProc = ifrit.Envoke(pinger)
			errChan = make(chan error)
		})

		Describe("Wait()", func() {
			BeforeEach(func() {
				go func() {
					errChan <- pingProc.Wait()
				}()
				go func() {
					errChan <- pingProc.Wait()
				}()
			})

			Context("when the process exits", func() {
				BeforeEach(func() {
					go func() {
						<-pinger
					}()
				})

				It("returns the run result upon completion", func() {
					err1 := <-errChan
					err2 := <-errChan
					Ω(err1).Should(Equal(PingerExitedFromPing))
					Ω(err2).Should(Equal(PingerExitedFromPing))
				})
			})
		})

		Describe("Signal()", func() {
			BeforeEach(func() {
				pingProc.Signal(os.Kill)
			})
			It("sends the signal to the runner", func() {
				err := pingProc.Wait()
				Ω(err).Should(Equal(PingerExitedFromSignal))
			})
		})
	})
})
