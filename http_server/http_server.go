package http_server

import (
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/tedsuo/ifrit"
)

type httpServer struct {
	address string
	handler http.Handler

	connectionWaitGroup *sync.WaitGroup
	connections         map[net.Conn]struct{}
	connectionsMu       *sync.Mutex
	stoppingChan        chan struct{}
}

func New(address string, handler http.Handler) ifrit.Runner {
	return &httpServer{
		address: address,
		handler: handler,
	}
}

func (s *httpServer) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	s.connectionWaitGroup = new(sync.WaitGroup)
	s.connectionsMu = new(sync.Mutex)
	s.connections = make(map[net.Conn]struct{})
	s.stoppingChan = make(chan struct{})

	server := http.Server{
		Handler: s.handler,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				s.connectionWaitGroup.Add(1)
				s.addConnection(conn)

			case http.StateHijacked, http.StateClosed:
				s.removeConnection(conn)
				s.connectionWaitGroup.Done()
			}
		},
	}

	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- server.Serve(listener)
	}()

	close(ready)

	for {
		select {
		case err = <-serverErrChan:
			return err

		case sig := <-signals:
			close(s.stoppingChan)

			listener.Close()

			s.connectionsMu.Lock()
			for c := range s.connections {
				c.Close()
			}
			s.connectionsMu.Unlock()

			if sig != os.Kill {
				s.connectionWaitGroup.Wait()
			}
			return nil
		}
	}
}

func (s *httpServer) addConnection(conn net.Conn) {
	select {
	case <-s.stoppingChan:
		conn.Close()
	default:
		s.connectionsMu.Lock()
		s.connections[conn] = struct{}{}
		s.connectionsMu.Unlock()
	}
}

func (s *httpServer) removeConnection(conn net.Conn) {
	s.connectionsMu.Lock()
	delete(s.connections, conn)
	s.connectionsMu.Unlock()
}
