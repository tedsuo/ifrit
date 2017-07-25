package grpc_server_test

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grpc_server"
	"golang.org/x/net/context"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"os"
	"path"
)

var _ = Describe("GRPCServer", func() {
	var (
		listenAddress string
		runner        ifrit.Runner
		serverProcess ifrit.Process
		tlsConfig     *tls.Config
	)

	BeforeEach(func() {
		var err error

		basePath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "tedsuo", "ifrit", "http_server", "test_certs")
		certFile := path.Join(basePath, "server.crt")
		keyFile := path.Join(basePath, "server.key")

		tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
		Expect(err).NotTo(HaveOccurred())

		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{tlsCert},
		}

		listenAddress = fmt.Sprintf("localhost:%d", 10000+GinkgoParallelNode())
	})

	Context("given an instatiated runner", func() {
		BeforeEach(func() {
			runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, helloworld.RegisterGreeterServer)
		})
		JustBeforeEach(func() {
			serverProcess = ginkgomon.Invoke(runner)
		})

		AfterEach(func() {
			ginkgomon.Kill(serverProcess)
		})

		It("serves on the listen address", func() {
			conn, err := grpc.Dial(listenAddress, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
			Expect(err).NotTo(HaveOccurred())

			helloClient := helloworld.NewGreeterClient(conn)
			_, err = helloClient.SayHello(context.Background(), &helloworld.HelloRequest{Name: "Fred"})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the server trys to listen on a busy port", func() {
			var alternateRunner ifrit.Runner

			BeforeEach(func() {
				alternateRunner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, helloworld.RegisterGreeterServer)
			})

			It("exits with an error", func() {
				var err error
				process := ifrit.Background(alternateRunner)
				Eventually(process.Wait()).Should(Receive(&err))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("when there is no tls config", func() {
		BeforeEach(func() {
			runner = grpc_server.NewGRPCServer(listenAddress, nil, &server{}, helloworld.RegisterGreeterServer)
		})
		JustBeforeEach(func() {
			serverProcess = ginkgomon.Invoke(runner)
		})

		AfterEach(func() {
			ginkgomon.Kill(serverProcess)
		})

		It("serves on the listen address", func() {
			conn, err := grpc.Dial(listenAddress, grpc.WithInsecure())
			Expect(err).NotTo(HaveOccurred())

			helloClient := helloworld.NewGreeterClient(conn)
			_, err = helloClient.SayHello(context.Background(), &helloworld.HelloRequest{Name: "Fred"})
			Expect(err).NotTo(HaveOccurred())
		})

	})

	Context("when the inputs to NewGRPCServer are incorrectly typed", func() {
		It("panics", func() {
			Expect(func() {
				grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, 42)
			}).To(Panic())
			Expect(func() {
				grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func() {})
			}).To(Panic())
			Expect(func() {
				grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func(a, b int) {})
			}).To(Panic())
			Expect(func() {
				grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func(a *grpc.Server, b int) {})
			}).To(Panic())
			Expect(func() {
				f := func(a *grpc.Server, b helloworld.GreeterServer) error { return nil }
				grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, f)
			}).To(Panic())
			Expect(func() {
				grpc_server.NewGRPCServer(listenAddress, tlsConfig, &notServer{}, helloworld.RegisterGreeterServer)
			}).To(Panic())
			Expect(func() {
				grpc_server.NewGRPCServer(listenAddress, tlsConfig, 42, helloworld.RegisterGreeterServer)
			}).To(Panic())
		})
	})
})

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: "Hello " + in.Name}, nil
}

// notServer doesn't implement anything
type notServer struct{}
