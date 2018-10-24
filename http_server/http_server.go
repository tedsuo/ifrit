package http_server

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/tedsuo/ifrit"
)

const (
	TCP        = "tcp"
	UNIX       = "unix"
	NO_TIMEOUT = 0 * time.Second
)

type httpServer struct {
	protocol string
	address  string
	handler  http.Handler

	readTimeout  time.Duration
	writeTimeout time.Duration

	connectionWaitGroup   *sync.WaitGroup
	inactiveConnections   map[net.Conn]struct{}
	inactiveConnectionsMu *sync.Mutex
	stoppingChan          chan struct{}

	tlsConfig *tls.Config
}

func newServerWithListener(protocol, address string, handler http.Handler, tlsConfig *tls.Config) ifrit.Runner {
	return &httpServer{
		address:   address,
		handler:   handler,
		tlsConfig: tlsConfig,
		protocol:  protocol,
	}
}

func newServerWithTimeout(protocol, address string, handler http.Handler, tlsConfig *tls.Config, readTimeout, writeTimeout time.Duration) ifrit.Runner {
	return &httpServer{
		address:      address,
		handler:      handler,
		tlsConfig:    tlsConfig,
		protocol:     protocol,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

func NewUnixServer(address string, handler http.Handler) ifrit.Runner {
	return newServerWithListener(UNIX, address, handler, nil)
}

func NewServerWithTimeout(address string, handler http.Handler, readTimeout, writeTimeout time.Duration) ifrit.Runner {
	return newServerWithTimeout(TCP, address, handler, nil, readTimeout, writeTimeout)
}

func New(address string, handler http.Handler) ifrit.Runner {
	return newServerWithListener(TCP, address, handler, nil)
}

func NewUnixTLSServer(address string, handler http.Handler, tlsConfig *tls.Config) ifrit.Runner {
	return newServerWithListener(UNIX, address, handler, tlsConfig)
}

func NewUnixTLSServerWithTimeout(address string, handler http.Handler, tlsConfig *tls.Config, readTimeout, writeTimeout time.Duration) ifrit.Runner {
	return newServerWithTimeout(UNIX, address, handler, tlsConfig, readTimeout, writeTimeout)
}

func NewTLSServer(address string, handler http.Handler, tlsConfig *tls.Config) ifrit.Runner {
	return newServerWithListener(TCP, address, handler, tlsConfig)
}

func NewTLSServerWithTimeout(address string, handler http.Handler, tlsConfig *tls.Config, readTimeout, writeTimeout time.Duration) ifrit.Runner {
	return newServerWithTimeout(TCP, address, handler, tlsConfig, readTimeout, writeTimeout)
}

func (s *httpServer) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	s.connectionWaitGroup = new(sync.WaitGroup)
	s.inactiveConnectionsMu = new(sync.Mutex)
	s.inactiveConnections = make(map[net.Conn]struct{})
	s.stoppingChan = make(chan struct{})

	connCountCh := make(chan int)

	server := http.Server{
		Handler:   s.handler,
		TLSConfig: s.tlsConfig,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				connCountCh <- 1
				s.addInactiveConnection(conn)

			case http.StateIdle:
				s.addInactiveConnection(conn)

			case http.StateActive:
				s.removeInactiveConnection(conn)

			case http.StateHijacked, http.StateClosed:
				s.removeInactiveConnection(conn)
				connCountCh <- -1
			}
		},
	}

	if s.readTimeout != NO_TIMEOUT {
		server.ReadTimeout = s.readTimeout
	}

	if s.writeTimeout != NO_TIMEOUT {
		server.WriteTimeout = s.writeTimeout
	}

	listener, err := s.getListener(server.TLSConfig)
	if err != nil {
		return err
	}

	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- server.Serve(listener)
	}()

	close(ready)

	connCount := 0
	for {
		select {
		case err = <-serverErrChan:
			return err

		case delta := <-connCountCh:
			connCount += delta

		case <-signals:
			close(s.stoppingChan)

			listener.Close()

			s.inactiveConnectionsMu.Lock()
			for c := range s.inactiveConnections {
				c.Close()
			}
			s.inactiveConnectionsMu.Unlock()

			for connCount != 0 {
				delta := <-connCountCh
				connCount += delta
			}

			return nil
		}
	}
}

func (s *httpServer) getListener(tlsConfig *tls.Config) (net.Listener, error) {
	listener, err := net.Listen(s.protocol, s.address)
	if err != nil {
		return nil, err
	}
	if tlsConfig == nil {
		return listener, nil
	}
	switch s.protocol {
	case TCP:
		listener = tls.NewListener(tcpKeepAliveListener{listener.(*net.TCPListener)}, tlsConfig)
	default:
		listener = tls.NewListener(listener, tlsConfig)
	}

	return listener, nil
}

func (s *httpServer) addInactiveConnection(conn net.Conn) {
	select {
	case <-s.stoppingChan:
		conn.Close()
	default:
		s.inactiveConnectionsMu.Lock()
		s.inactiveConnections[conn] = struct{}{}
		s.inactiveConnectionsMu.Unlock()
	}
}

func (s *httpServer) removeInactiveConnection(conn net.Conn) {
	s.inactiveConnectionsMu.Lock()
	delete(s.inactiveConnections, conn)
	s.inactiveConnectionsMu.Unlock()
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
