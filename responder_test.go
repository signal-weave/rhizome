package rhizome

import (
	"bytes"
	"errors"
	"sync"
	"testing"
	"time"
)

// -------tests-----------------------------------------------------------------

func TestConnResponder_RemoteAddr(t *testing.T) {
	fc := newFakeConn("10.1.2.3:4444")
	cr := &ConnResponder{C: fc}

	got := cr.RemoteAddr()
	want := "10.1.2.3:4444"
	if got != want {
		t.Fatalf("RemoteAddr() = %q, want %q", got, want)
	}
}

func TestConnResponder_Write_WritesBytesAndReturnsNil(t *testing.T) {
	fc := newFakeConn("1.2.3.4:5")
	cr := &ConnResponder{C: fc}

	data := []byte("hello world")
	if err := cr.Write(data); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if !bytes.Contains(fc.buf.Bytes(), data) {
		t.Fatalf("underlying conn buffer does not contain written bytes; got=%q", fc.buf.Bytes())
	}
}

func TestConnResponder_Write_SerializesConcurrentWrites(t *testing.T) {
	fc := newFakeConn("1.2.3.4:5")
	// Add a small delay inside Write so that overlapping calls would overlap
	// if the ConnResponder didn't serialize them.
	fc.writeDelay = 10 * time.Millisecond

	cr := &ConnResponder{C: fc}

	var wg sync.WaitGroup
	chunks := [][]byte{
		[]byte("CHUNK-1\n"),
		[]byte("CHUNK-2\n"),
		[]byte("CHUNK-3\n"),
		[]byte("CHUNK-4\n"),
		[]byte("CHUNK-5\n"),
	}

	wg.Add(len(chunks))
	for _, b := range chunks {
		b := b
		go func() {
			defer wg.Done()
			if err := cr.Write(b); err != nil {
				t.Errorf("Write() unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	// If the internal mutex works, maxSeen should be 1 and overlap false.
	if fc.maxSeen != 1 || fc.overlap {
		t.Fatalf("writes overlapped: maxSeen=%d overlap=%v; mutex not enforcing serialization", fc.maxSeen, fc.overlap)
	}

	// Sanity: total bytes should equal the sum of chunks.
	total := 0
	for _, b := range chunks {
		total += len(b)
	}
	if fc.buf.Len() != total {
		t.Fatalf("buffer length=%d, want %d", fc.buf.Len(), total)
	}
	// Each chunk should appear at least once (order is not asserted).
	full := fc.buf.Bytes()
	for _, b := range chunks {
		if !bytes.Contains(full, b) {
			t.Fatalf("buffer does not contain chunk %q; got=%q", b, full)
		}
	}
}

func TestConnResponder_Write_PropagatesUnderlyingError(t *testing.T) {
	fc := newFakeConn("1.2.3.4:5")
	fc.writeErr = errors.New("boom")
	cr := &ConnResponder{C: fc}

	err := cr.Write([]byte("data"))
	if err == nil {
		t.Fatalf("Write() expected error to propagate from underlying conn")
	}
	if err.Error() != "boom" {
		t.Fatalf("Write() returned %v, want %v", err, fc.writeErr)
	}
}
