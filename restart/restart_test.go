package restart_test

import (
	"os"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/fake_runner"
	"github.com/tedsuo/ifrit/restart"
)

var _ = Describe("Restart", func() {
	var loadRunner *fake_runner.TestRunner
	var testRunner *fake_runner.TestRunner
	var restarter *restart.Restarter
	var process ifrit.Process

	BeforeEach(func() {
		testRunner = fake_runner.NewTestRunner()
		loadRunner = fake_runner.NewTestRunner()
		restarter = &restart.Restarter{
			Runner: testRunner,
			Load: func(runner ifrit.Runner, err error) ifrit.Runner {
				return loadRunner
			},
		}
	})

	JustBeforeEach(func() {
		process = ifrit.Background(restarter)
	})

	AfterEach(func() {
		testRunner.EnsureExit()
		loadRunner.EnsureExit()
		process.Signal(os.Kill)
		Eventually(process.Wait()).Should(Receive())
	})

	Describe("Process Behavior", func() {

		It("waits for the internal runner to be ready", func() {
			Consistently(process.Ready()).ShouldNot(BeClosed())
			testRunner.TriggerReady()
			Eventually(process.Ready()).Should(BeClosed())
		})
	})

	Describe("Load", func() {

		Context("when load returns a runner", func() {
			It("executes the returned Runner", func() {
				testRunner.TriggerExit(nil)
				loadRunner.TriggerExit(nil)
			})
		})

		Context("when load returns nil", func() {
			BeforeEach(func() {
				restarter.Load = func(runner ifrit.Runner, err error) ifrit.Runner {
					return nil
				}
			})

			It("exits after running the initial Runner", func() {
				testRunner.TriggerExit(nil)
				Eventually(process.Wait()).Should(Receive(BeNil()))
			})
		})

		Context("when the load callback is nil", func() {
			BeforeEach(func() {
				restarter.Load = nil
			})

			It("exits with NoLoadCallback error", func() {
				Eventually(process.Wait()).Should(Receive(Equal(restart.ErrNoLoadCallback)))
			})
		})
	})
})
