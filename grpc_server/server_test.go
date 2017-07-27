package grpc_server_test

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grpc_server"
	"golang.org/x/net/context"
	"google.golang.org/grpc/examples/helloworld/helloworld"
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

	Context("when the inputs to NewGRPCServer are invalid", func() {
		var (
			err error
		)
		JustBeforeEach(func() {
			process := ifrit.Background(runner)
			Eventually(process.Wait()).Should(Receive(&err))
			Expect(err).To(HaveOccurred())
		})

		Context("when the registrar is an integer", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, 42)
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("should be func but is int"))
			})
		})

		Context("when the registrar is nil", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, nil)
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("`serverRegistrar` and `handler` must be non nil"))
			})
		})

		Context("when the registrar is nil", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, nil, helloworld.RegisterGreeterServer)
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("`serverRegistrar` and `handler` must be non nil"))
			})
		})

		Context("when the registrar is an empty func", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func() {})
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("should have 2 parameters but it has 0 parameters"))
			})
		})

		Context("when the registrar has bad parameters", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func(a, b int) {})
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("first parameter must be `*grpc.Server` but is int"))
			})
		})

		Context("when the registrar's first parameter is bad", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func(a, b int) {})
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("first parameter must be `*grpc.Server` but is int"))
			})
		})

		Context("when the registrar's second parameter is not an interface", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func(a *grpc.Server, b int) {})
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("is not implemented by `handler`"))
			})
		})

		Context("when the registrar's second parameter is not implemented", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, func(a *grpc.Server, b testInterface) {})
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("is not implemented by `handler`"))
			})
		})

		Context("when the handler is a *struct but doesn't implement the registrar's second parameter", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &notServer{}, helloworld.RegisterGreeterServer)
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("is not implemented by `handler`"))
			})
		})

		Context("when the handler is a int but doesn't implement the registrar's second parameter", func() {
			BeforeEach(func() {
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, 42, helloworld.RegisterGreeterServer)
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("is not implemented by `handler`"))
			})
		})

		Context("when the registrar returns a value", func() {
			BeforeEach(func() {
				f := func(a *grpc.Server, b helloworld.GreeterServer) error { return nil }
				runner = grpc_server.NewGRPCServer(listenAddress, tlsConfig, &server{}, f)
			})
			It("fails", func() {
				Expect(err.Error()).To(ContainSubstring("should return no value but it returns 1 value"))
			})
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

type testInterface interface {
	something(a int) int
}
