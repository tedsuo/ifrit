package grouper_test

import (
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/fake_runner"
	"github.com/tedsuo/ifrit/grouper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ordered Group", func() {
	var (
		started chan struct{}

		groupRunner  ifrit.Runner
		groupProcess ifrit.Process
		members      grouper.Members

		childRunner1 *fake_runner.TestRunner
		childRunner2 *fake_runner.TestRunner
		childRunner3 *fake_runner.TestRunner

		Δ time.Duration = 10 * time.Millisecond
	)

	BeforeEach(func() {
		childRunner1 = fake_runner.NewTestRunner()
		childRunner2 = fake_runner.NewTestRunner()
		childRunner3 = fake_runner.NewTestRunner()

		members = grouper.Members{
			{"child1", childRunner1},
			{"child2", childRunner2},
			{"child3", childRunner3},
		}

		groupRunner = grouper.NewOrdered(os.Interrupt, members)
	})

	AfterEach(func() {
		childRunner1.EnsureExit()
		childRunner2.EnsureExit()
		childRunner3.EnsureExit()

		Eventually(started).Should(BeClosed())
		groupProcess.Signal(os.Kill)
		Eventually(groupProcess.Wait()).Should(Receive())
	})

	Describe("Start", func() {
		BeforeEach(func() {
			started = make(chan struct{})
			go func() {
				groupProcess = ifrit.Envoke(groupRunner)
				close(started)
			}()
		})

		It("runs the first runner, then the second, then becomes ready", func() {
			Eventually(childRunner1.RunCallCount).Should(Equal(1))
			Consistently(childRunner2.RunCallCount, Δ).Should(BeZero())
			Consistently(started, Δ).ShouldNot(BeClosed())

			childRunner1.TriggerReady()

			Eventually(childRunner2.RunCallCount).Should(Equal(1))
			Consistently(childRunner3.RunCallCount, Δ).Should(BeZero())
			Consistently(started, Δ).ShouldNot(BeClosed())

			childRunner2.TriggerReady()

			Eventually(childRunner3.RunCallCount).Should(Equal(1))
			Consistently(started, Δ).ShouldNot(BeClosed())

			childRunner3.TriggerReady()

			Eventually(started).Should(BeClosed())
		})

		Describe("when all the runners are ready", func() {
			var (
				signal1 <-chan os.Signal
				signal2 <-chan os.Signal
				signal3 <-chan os.Signal
			)

			BeforeEach(func() {
				signal1 = childRunner1.WaitForCall()
				childRunner1.TriggerReady()
				signal2 = childRunner2.WaitForCall()
				childRunner2.TriggerReady()
				signal3 = childRunner3.WaitForCall()
				childRunner3.TriggerReady()

				Eventually(started).Should(BeClosed())
			})

			Describe("when it receives a signal", func() {
				BeforeEach(func() {
					groupProcess.Signal(syscall.SIGUSR2)
				})

				It("sends the signal to all child runners", func() {
					Eventually(signal1).Should(Receive(Equal(syscall.SIGUSR2)))
					Eventually(signal2).Should(Receive(Equal(syscall.SIGUSR2)))
					Eventually(signal3).Should(Receive(Equal(syscall.SIGUSR2)))
				})

				It("doesn't send any more signals to remaining child processes", func() {
					Eventually(signal3).Should(Receive(Equal(syscall.SIGUSR2)))
					childRunner2.TriggerExit(nil)
					Consistently(signal3).ShouldNot(Receive())
				})
			})

			Describe("when a process exits cleanly", func() {
				BeforeEach(func() {
					childRunner1.TriggerExit(nil)
				})

				It("sends an interrupt signal to the other processes", func() {
					Eventually(signal2).Should(Receive(Equal(os.Interrupt)))
					Eventually(signal3).Should(Receive(Equal(os.Interrupt)))
				})

				It("does not exit", func() {
					Consistently(groupProcess.Wait(), Δ).ShouldNot(Receive())
				})

				Describe("when another process exits", func() {
					BeforeEach(func() {
						childRunner2.TriggerExit(nil)
					})

					It("doesn't send any more signals to remaining child processes", func() {
						Eventually(signal3).Should(Receive(Equal(os.Interrupt)))
						Consistently(signal3).ShouldNot(Receive())
					})
				})

				Describe("when all of the processes have exited cleanly", func() {
					BeforeEach(func() {
						childRunner2.TriggerExit(nil)
						childRunner3.TriggerExit(nil)
					})

					It("exits cleanly", func() {
						Eventually(groupProcess.Wait()).Should(Receive(BeNil()))
					})
				})

				Describe("when one of the processes exits with an error", func() {
					BeforeEach(func() {
						childRunner2.TriggerExit(errors.New("Fail"))
						childRunner3.TriggerExit(nil)
					})

					It("returns an error indicating which child processes failed", func() {
						var err error
						Eventually(groupProcess.Wait()).Should(Receive(&err))
						Ω(err).Should(Equal(grouper.ErrorTrace{
							{grouper.Member{"child1", childRunner1}, nil},
							{grouper.Member{"child2", childRunner2}, errors.New("Fail")},
							{grouper.Member{"child3", childRunner3}, nil},
						}))
					})
				})
			})
		})

		Describe("Failed start", func() {
			BeforeEach(func() {
				signal1 := childRunner1.WaitForCall()
				childRunner1.TriggerReady()
				childRunner2.TriggerExit(errors.New("Fail"))
				Eventually(signal1).Should(Receive(Equal(os.Interrupt)))
				childRunner1.TriggerExit(nil)
				Eventually(started).Should(BeClosed())
			})

			It("exits without starting further processes", func() {
				var err error

				Eventually(groupProcess.Wait()).Should(Receive(&err))
				Ω(err).Should(Equal(grouper.ErrorTrace{
					{grouper.Member{"child2", childRunner2}, errors.New("Fail")},
					{grouper.Member{"child1", childRunner1}, nil},
				}))
			})
		})
	})
})
