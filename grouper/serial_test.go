package grouper_test

import (
	"errors"
	"os"
	"time"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/fake_runner"
	"github.com/tedsuo/ifrit/grouper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Serial Group", func() {
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

		groupRunner = grouper.NewSerial(members)

	})

	AfterEach(func() {
		childRunner1.EnsureExit()
		childRunner2.EnsureExit()
		childRunner3.EnsureExit()

		Eventually(started).Should(BeClosed())
		groupProcess.Signal(os.Kill)
		Eventually(groupProcess.Wait()).Should(Receive())
	})

	Describe("Invoke", func() {
		BeforeEach(func() {
			started = make(chan struct{})
			go func() {
				groupProcess = ifrit.Envoke(groupRunner)
				close(started)
			}()
		})

		Describe("serial execution", func() {
			It("executes the runners one at a time", func() {
				Eventually(childRunner1.RunCallCount).Should(Equal(1))
				Consistently(childRunner2.RunCallCount, Δ).Should(BeZero())
				Consistently(started, Δ).ShouldNot(BeClosed())

				childRunner1.TriggerReady()
				Consistently(childRunner2.RunCallCount, Δ).Should(BeZero())
				Consistently(childRunner3.RunCallCount, Δ).Should(BeZero())
				Consistently(started, Δ).ShouldNot(BeClosed())

				childRunner1.TriggerExit(nil)
				Eventually(childRunner2.RunCallCount).Should(Equal(1))
				Consistently(childRunner3.RunCallCount, Δ).Should(BeZero())
				Consistently(started, Δ).ShouldNot(BeClosed())

				childRunner2.TriggerReady()
				Consistently(childRunner3.RunCallCount, Δ).Should(BeZero())
				Consistently(started, Δ).ShouldNot(BeClosed())

				childRunner2.TriggerExit(nil)
				Eventually(childRunner3.RunCallCount).Should(Equal(1))
				Consistently(started, Δ).ShouldNot(BeClosed())

				childRunner3.TriggerReady()
				Consistently(started, Δ).ShouldNot(BeClosed())

				childRunner3.TriggerExit(nil)
				Eventually(started).Should(BeClosed())
			})
		})

		Describe("when all of the processes have exited cleanly", func() {
			BeforeEach(func() {
				childRunner1.TriggerReady()
				childRunner1.TriggerExit(nil)
				childRunner2.TriggerReady()
				childRunner2.TriggerExit(nil)
				childRunner3.TriggerReady()
				childRunner3.TriggerExit(nil)
				Eventually(started).Should(BeClosed())
			})

			It("exits cleanly", func() {
				Eventually(groupProcess.Wait()).Should(Receive(BeNil()))
			})
		})

		Describe("Failed start", func() {
			BeforeEach(func() {
				childRunner1.TriggerReady()
				childRunner1.TriggerExit(nil)
				childRunner2.TriggerReady()
				childRunner2.TriggerExit(errors.New("Fail"))
				Eventually(started).Should(BeClosed())
			})

			It("exits without starting further processes", func() {
				var err error

				Eventually(groupProcess.Wait()).Should(Receive(&err))
				Ω(err).Should(Equal(grouper.ErrorTrace{
					{grouper.Member{"child1", childRunner1}, nil},
					{grouper.Member{"child2", childRunner2}, errors.New("Fail")},
				}))
			})
		})
	})
})
