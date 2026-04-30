package netmux

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

var http2Preface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")

type Router func(prefix []byte) string

type Mux struct {
	base    net.Listener
	router  Router
	timeout time.Duration

	mu        sync.Mutex
	closed    bool
	listeners map[string]*subListener
}

func New(base net.Listener, router Router) *Mux {
	return &Mux{
		base:      base,
		router:    router,
		timeout:   2 * time.Second,
		listeners: map[string]*subListener{},
	}
}

func (m *Mux) Listener(name string) net.Listener {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing := m.listeners[name]; existing != nil {
		return existing
	}
	ln := &subListener{
		parent: m,
		name:   name,
		conns:  make(chan net.Conn, 128),
	}
	m.listeners[name] = ln
	return ln
}

func (m *Mux) Serve() error {
	for {
		conn, err := m.base.Accept()
		if err != nil {
			if m.isClosed() {
				return net.ErrClosed
			}
			return err
		}
		go m.route(conn)
	}
}

func (m *Mux) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	listeners := make([]*subListener, 0, len(m.listeners))
	for _, ln := range m.listeners {
		listeners = append(listeners, ln)
	}
	m.mu.Unlock()

	err := m.base.Close()
	for _, ln := range listeners {
		ln.close()
	}
	return err
}

func (m *Mux) Addr() net.Addr {
	return m.base.Addr()
}

func (m *Mux) route(conn net.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(m.timeout))
	reader := bufio.NewReader(conn)
	route, err := m.detectRoute(reader)
	if err != nil {
		_ = conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	ln := m.get(route)
	if ln == nil {
		_ = conn.Close()
		return
	}
	ln.deliver(&bufferedConn{Conn: conn, reader: reader})
}

func (m *Mux) detectRoute(reader *bufio.Reader) (string, error) {
	prefix, err := reader.Peek(1)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(http2Preface, prefix) {
		return m.router(prefix), nil
	}

	for size := 3; size <= len(http2Preface); size++ {
		prefix, err = reader.Peek(size)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", err
			}
			if reader.Buffered() > 0 {
				return m.router(prefix), nil
			}
			return "", err
		}
		if !bytes.HasPrefix(http2Preface, prefix) {
			return m.router(prefix), nil
		}
		if bytes.Equal(prefix, http2Preface) {
			return m.router(prefix), nil
		}
	}
	return m.router(prefix), nil
}

func (m *Mux) get(name string) *subListener {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listeners[name]
}

func (m *Mux) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func IsGRPCPrefix(prefix []byte) bool {
	return bytes.HasPrefix(http2Preface, prefix) || bytes.HasPrefix(prefix, http2Preface)
}

type subListener struct {
	parent *Mux
	name   string
	conns  chan net.Conn

	once sync.Once
}

func (l *subListener) Accept() (net.Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, net.ErrClosed
	}
	return conn, nil
}

func (l *subListener) Close() error {
	l.close()
	return nil
}

func (l *subListener) Addr() net.Addr {
	return l.parent.Addr()
}

func (l *subListener) close() {
	l.once.Do(func() {
		close(l.conns)
	})
}

func (l *subListener) deliver(conn net.Conn) {
	defer func() {
		if recover() != nil {
			_ = conn.Close()
		}
	}()
	l.conns <- conn
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	if c.reader != nil && c.reader.Buffered() > 0 {
		return c.reader.Read(p)
	}
	if c.Conn == nil {
		return 0, errors.New("nil connection")
	}
	return c.Conn.Read(p)
}
