package rhizome

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"time"
)

type stubAddr struct{ s string }

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return a.s }

// fakeConn is a minimal in-memory net.Conn used for tests.
// It captures writes, allows injecting errors, and can detect overlapping
// writes.
type fakeConn struct {
	buf bytes.Buffer

	remote net.Addr
	local  net.Addr

	// If set, Write returns this error after writing 0 bytes.
	writeErr error

	// Concurrency detection
	muActive sync.Mutex
	active   int
	maxSeen  int
	overlap  bool

	// Optional artificial delay in Write to help surface overlap if locks are
	// missing.
	writeDelay time.Duration

	closed bool
	mu     sync.Mutex
}

func newFakeConn(remoteStr string) *fakeConn {
	return &fakeConn{
		remote: stubAddr{s: remoteStr},
		local:  stubAddr{s: "127.0.0.1:0"},
	}
}

func (c *fakeConn) Read(p []byte) (int, error)         { return 0, errors.New("unimplemented for tests") }
func (c *fakeConn) Close() error                       { c.mu.Lock(); defer c.mu.Unlock(); c.closed = true; return nil }
func (c *fakeConn) LocalAddr() net.Addr                { return c.local }
func (c *fakeConn) RemoteAddr() net.Addr               { return c.remote }
func (c *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

func (c *fakeConn) Write(p []byte) (int, error) {
	// Track overlap: if another Write is running at the same time, mark
	// overlap=true.
	c.muActive.Lock()
	c.active++
	if c.active > c.maxSeen {
		c.maxSeen = c.active
	}
	if c.active > 1 {
		c.overlap = true
	}
	c.muActive.Unlock()

	defer func() {
		c.muActive.Lock()
		c.active--
		c.muActive.Unlock()
	}()

	if c.writeErr != nil {
		return 0, c.writeErr
	}

	if c.writeDelay > 0 {
		time.Sleep(c.writeDelay)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}
