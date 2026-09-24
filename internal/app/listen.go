package app

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"syscall"
)

func listen(host string, port int) (net.Listener, error) {
	if host == "" || host == "localhost" {
		return listenLoopback(port)
	}
	return net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
}

func listenLoopback(port int) (net.Listener, error) {
	for attempt := 0; ; attempt++ {
		v4, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			return nil, err
		}
		bound := v4.Addr().(*net.TCPAddr).Port
		v6, err := net.Listen("tcp6", net.JoinHostPort("::1", strconv.Itoa(bound)))
		switch {
		case err == nil:
			return newMultiListener(v4, v6), nil
		case !errors.Is(err, syscall.EADDRINUSE):
			return v4, nil
		case port == 0 && attempt < 10:
			_ = v4.Close()
		default:
			_ = v4.Close()
			return nil, err
		}
	}
}

type multiListener struct {
	listeners []net.Listener
	conns     chan net.Conn
	errs      chan error
	done      chan struct{}
	once      sync.Once
}

func newMultiListener(listeners ...net.Listener) *multiListener {
	m := &multiListener{
		listeners: listeners,
		conns:     make(chan net.Conn),
		errs:      make(chan error),
		done:      make(chan struct{}),
	}
	for _, l := range listeners {
		go m.accept(l)
	}
	return m
}

func (m *multiListener) accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case m.errs <- err:
			case <-m.done:
				return
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		select {
		case m.conns <- conn:
		case <-m.done:
			_ = conn.Close()
			return
		}
	}
}

func (m *multiListener) Accept() (net.Conn, error) {
	select {
	case conn := <-m.conns:
		return conn, nil
	case <-m.done:
		return nil, net.ErrClosed
	case err := <-m.errs:
		return nil, err
	}
}

func (m *multiListener) Close() error {
	var err error
	m.once.Do(func() {
		close(m.done)
		for _, l := range m.listeners {
			err = errors.Join(err, l.Close())
		}
	})
	return err
}

func (m *multiListener) Addr() net.Addr { return m.listeners[0].Addr() }
