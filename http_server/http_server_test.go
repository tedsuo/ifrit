package http_server_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/http_server/unix_transport"
)

var _ = Describe("HttpServer", func() {
	var (
		address            string
		server             ifrit.Runner
		startedRequestChan chan struct{}
		finishRequestChan  chan struct{}

		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedRequestChan <- struct{}{}
			<-finishRequestChan
			w.Write([]byte("yo")) //nolint:errcheck
		})
	)

	BeforeEach(func() {
		startedRequestChan = make(chan struct{}, 1)
		finishRequestChan = make(chan struct{}, 1)
		port := 8000 + GinkgoParallelProcess()
		address = fmt.Sprintf("127.0.0.1:%d", port)
	})

	Describe("Invoke", func() {
		var process ifrit.Process
		var socketPath string

		Context("when the server is started with a different net Protocol.", func() {
			var tmpdir string

			BeforeEach(func() {
				unixHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("yo")) //nolint:errcheck
				})
				var err error
				tmpdir, err = os.MkdirTemp(os.TempDir(), "ifrit-server-test")
				Ω(err).ShouldNot(HaveOccurred())

				socketPath = path.Join(tmpdir, "ifrit.sock")
				Ω(err).ShouldNot(HaveOccurred())
				server = http_server.NewUnixServer(socketPath, unixHandler)
				process = ifrit.Invoke(server)
			})

			AfterEach(func() {
				process.Signal(syscall.SIGINT)
				Eventually(process.Wait()).Should(Receive())
			})

			It("serves requests with the given handler", func() {
				resp, err := httpGetUnix("unix://"+socketPath+"/", socketPath)

				Ω(err).ShouldNot(HaveOccurred())
				body, err := io.ReadAll(resp.Body)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(body)).Should(Equal("yo"))
			})
		})

		Context("when the server starts successfully", func() {
			BeforeEach(func() {
				server = http_server.New(address, handler)
				process = ifrit.Invoke(server)
			})

			AfterEach(func() {
				process.Signal(syscall.SIGINT)
				Eventually(process.Wait()).Should(Receive())
			})

			Context("and a request is in flight", func() {
				type httpResponse struct {
					response *http.Response
					err      error
				}
				var responses chan httpResponse

				BeforeEach(func() {
					responses = make(chan httpResponse, 1)
					go func() {
						response, err := httpGet("http://" + address)
						responses <- httpResponse{response, err}
						close(responses)
					}()
					<-startedRequestChan
				})

				AfterEach(func() {
					Eventually(responses).Should(BeClosed())
				})

				It("serves http requests with the given handler", func() {
					finishRequestChan <- struct{}{}

					var resp httpResponse
					Eventually(responses).Should(Receive(&resp))

					Ω(resp.err).ShouldNot(HaveOccurred())

					body, err := io.ReadAll(resp.response.Body)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(body)).Should(Equal("yo"))
				})

				Context("and it receives a signal", func() {
					BeforeEach(func() {
						process.Signal(syscall.SIGINT)
					})

					It("stops serving new http requests", func() {
						_, err := httpGet("http://" + address)
						Ω(err).Should(HaveOccurred())

						// make sure we exit
						finishRequestChan <- struct{}{}
					})

					It("does not return an error", func() {
						finishRequestChan <- struct{}{}
						err := <-process.Wait()
						Ω(err).ShouldNot(HaveOccurred())
					})

					It("does not exit until all outstanding requests are complete", func() {
						Consistently(process.Wait()).ShouldNot(Receive())
						finishRequestChan <- struct{}{}
						Eventually(process.Wait()).Should(Receive())
					})

					It("exits cleanly without error after graceful shutdown", func() {
						finishRequestChan <- struct{}{}
						var exitErr error
						Eventually(process.Wait()).Should(Receive(&exitErr))
						Ω(exitErr).ShouldNot(HaveOccurred())
					})
				})
			})
		})

		Context("when the server fails to start", func() {
			BeforeEach(func() {
				address = "127.0.0.1:80"
				server = http_server.New(address, handler)
			})

			It("returns an error", func() {
				process = ifrit.Invoke(server)
				err := <-process.Wait()
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("when the TLS server is started with a different net Protocol.", func() {
			var tlsConfig *tls.Config
			var tmpdir string
			var socketPath string

			BeforeEach(func() {
				basePath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "tedsuo", "ifrit", "http_server", "test_certs")
				certFile := path.Join(basePath, "server.crt")
				keyFile := path.Join(basePath, "server.key")

				tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
				Expect(err).NotTo(HaveOccurred())

				tlsConfig = &tls.Config{
					InsecureSkipVerify: true, // nolint:gosec
				}

				serverTlsConfig := &tls.Config{
					Certificates: []tls.Certificate{tlsCert},
				}

				unixHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("yo")) //nolint:errcheck
				})
				tmpdir, err = os.MkdirTemp(os.TempDir(), "ifrit-server-test")
				Ω(err).ShouldNot(HaveOccurred())

				socketPath = path.Join(tmpdir, "ifrit.sock")
				Ω(err).ShouldNot(HaveOccurred())
				server = http_server.NewUnixTLSServer(socketPath, unixHandler, serverTlsConfig)
				process = ifrit.Invoke(server)
			})
			AfterEach(func() {
				process.Signal(syscall.SIGINT)
				Eventually(process.Wait()).Should(Receive())
			})

			It("serves tls-secured http requests with the given handler", func() {

				resp, err := httpTLSGetUnix("unix://"+socketPath+"/", socketPath, tlsConfig)
				Ω(err).ShouldNot(HaveOccurred())
				body, err := io.ReadAll(resp.Body)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(body)).Should(Equal("yo"))
			})
		})

		Context("and it starts a server with TLS", func() {
			var tlsConfig *tls.Config
			type httpResponse struct {
				response *http.Response
				err      error
			}
			var responses chan httpResponse

			BeforeEach(func() {
				basePath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "tedsuo", "ifrit", "http_server", "test_certs")
				certFile := path.Join(basePath, "server.crt")
				keyFile := path.Join(basePath, "server.key")

				tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
				Expect(err).NotTo(HaveOccurred())

				tlsConfig = &tls.Config{
					InsecureSkipVerify: true, // nolint:gosec
				}

				serverTlsConfig := &tls.Config{
					Certificates: []tls.Certificate{tlsCert},
				}

				server = http_server.NewTLSServer(address, handler, serverTlsConfig)
				process = ifrit.Invoke(server)
			})

			AfterEach(func() {
				process.Signal(syscall.SIGINT)
				Eventually(process.Wait()).Should(Receive())
			})

			Context("and a valid, secure request is in flight", func() {
				BeforeEach(func() {
					responses = make(chan httpResponse, 1)
					go func() {
						response, err := httpTLSGet("https://"+address, tlsConfig)
						responses <- httpResponse{response, err}
						close(responses)
					}()
					Eventually(startedRequestChan).Should(Receive())
				})

				AfterEach(func() {
					Eventually(responses).Should(BeClosed())
				})

				It("serves tls-secured http requests with the given handler", func() {
					finishRequestChan <- struct{}{}

					var resp httpResponse
					Eventually(responses).Should(Receive(&resp))

					Ω(resp.err).ShouldNot(HaveOccurred())

					body, err := io.ReadAll(resp.response.Body)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(body)).Should(Equal("yo"))
				})
			})

			Context("and an insecure request is in flight", func() {
				BeforeEach(func() {
					responses = make(chan httpResponse, 1)
					go func() {
						response, err := httpGet("http://" + address)
						responses <- httpResponse{response, err}
						close(responses)
					}()
					Consistently(startedRequestChan).ShouldNot(Receive())
				})

				AfterEach(func() {
					Eventually(responses).Should(BeClosed())
				})

				It("rejects insecure http requests and receives an error", func() {
					finishRequestChan <- struct{}{}

					var resp httpResponse
					Eventually(responses).Should(Receive(&resp))

					Ω(resp.err).ShouldNot(HaveOccurred())
					Ω(resp.response.StatusCode).Should(Equal(http.StatusBadRequest))
				})
			})
		})
	})
})

// Subprocess regression test — catches os.Exit / log.Fatalf in library code
//
// This test re-executes the compiled test binary as a child process with
// IFRIT_HTTP_SERVER_SUBPROCESS=1. TestMain (in the suite file) intercepts that
// flag and runs the server lifecycle without starting Ginkgo. If the library
// ever calls log.Fatalf during shutdown, the child exits non-zero and this
// test fails with a clear message — rather than silently killing the Ginkgo
// parallel runner and producing the opaque "timed out waiting for all parallel
// procs to report back" failure that was seen in guardian.
var _ = Describe("HttpServer subprocess shutdown", func() {
	It("exits with status 0 after graceful shutdown (no os.Exit in library)", func() {
		cmd := exec.Command(os.Args[0], "-test.run=TestHttpServer", "-test.v=false")
		cmd.Env = append(os.Environ(), "IFRIT_HTTP_SERVER_SUBPROCESS=1")
		out, err := cmd.CombinedOutput()
		Ω(err).ShouldNot(HaveOccurred(),
			"subprocess exited non-zero — library may have called log.Fatalf or os.Exit:\n"+string(out))
	})
})

func httpGetUnix(url, socketPath string) (*http.Response, error) {
	client := http.Client{
		Transport: unix_transport.New(socketPath),
	}
	return client.Get(url)
}

func httpGet(url string) (*http.Response, error) {
	client := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	return client.Get(url)
}

func httpTLSGet(url string, tlsConfig *tls.Config) (*http.Response, error) {
	client := http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: tlsConfig,
		},
	}
	return client.Get(url)
}

func httpTLSGetUnix(url string, socketPath string, tlsConfig *tls.Config) (*http.Response, error) {
	client := http.Client{
		Transport: unix_transport.NewWithTLS(socketPath, tlsConfig),
	}
	return client.Get(url)
}
