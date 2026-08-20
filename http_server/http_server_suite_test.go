package http_server_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

// TestMain intercepts subprocess mode before Ginkgo starts. This lets a test
// re-exec the compiled test binary with IFRIT_HTTP_SERVER_SUBPROCESS=1 to
// verify that graceful shutdown does not call os.Exit. If the library ever
// calls log.Fatalf again, the subprocess exits non-zero and the parent test
// fails with a clear error rather than a Ginkgo parallel-process timeout.
func TestMain(m *testing.M) {
	if os.Getenv("IFRIT_HTTP_SERVER_SUBPROCESS") == "1" {
		runShutdownSubprocess()
		// runShutdownSubprocess always calls os.Exit; this line is unreachable.
		return
	}
	os.Exit(m.Run())
}

// freeAddr finds an available TCP address by briefly listening on port 0.
func freeAddr() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "freeAddr: %v\n", err)
		os.Exit(2)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func runShutdownSubprocess() {
	addr := freeAddr()

	// Use a handler that keeps a request open until signalled, mirroring the
	// production scenario that triggered the original log.Fatalf bug.
	started := make(chan struct{}, 1)
	finish := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-finish
		w.Write([]byte("ok")) //nolint:errcheck
	})

	server := http_server.New(addr, handler)
	process := ifrit.Invoke(server)

	// Fire off an in-flight request so shutdown must drain it.
	go func() { //nolint:errcheck
		http.Get("http://" + addr)
	}()
	<-started // wait until the handler is blocking

	// Signal graceful shutdown while the request is still in flight.
	process.Signal(syscall.SIGINT)
	close(finish) // let the handler complete

	err := <-process.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestHttpServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HttpServer Suite")
}
