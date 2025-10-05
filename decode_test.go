package rhizome

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// -------helpers---------------------------------------------------------------

func u16BE(n uint16) []byte {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], n)
	return tmp[:]
}

// ------readU8-----------------------------------------------------------------

func TestReadU8_Success(t *testing.T) {
	in := []byte{0x7F}
	var out uint8
	if err := readU8(bytes.NewReader(in), &out); err != nil {
		t.Fatalf("readU8 error: %v", err)
	}
	if out != 0x7F {
		t.Fatalf("readU8 got %d, want %d", out, 0x7F)
	}
}

func TestReadU8_EOF(t *testing.T) {
	var out uint8
	err := readU8(bytes.NewReader(nil), &out)
	if err == nil {
		t.Fatalf("readU8 expected error, got nil")
	}
}

// -------readStringU8----------------------------------------------------------

func TestReadStringU8_Empty(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	_ = buf.WriteByte(0) // u8 length = 0
	s, err := readStringU8(buf)
	if err != nil {
		t.Fatalf("readStringU8 error: %v", err)
	}
	if s != "" {
		t.Fatalf("readStringU8 got %q, want empty string", s)
	}
}

func TestReadStringU8_Success(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	_ = buf.WriteByte(3)      // len
	_, _ = buf.WriteString("abc")
	s, err := readStringU8(buf)
	if err != nil {
		t.Fatalf("readStringU8 error: %v", err)
	}
	if s != "abc" {
		t.Fatalf("readStringU8 got %q, want %q", s, "abc")
	}
}

func TestReadStringU8_ShortPayload(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	_ = buf.WriteByte(4)      // claims 4 bytes
	_, _ = buf.WriteString("ab") // only 2 provided
	_, err := readStringU8(buf)
	if err == nil {
		t.Fatalf("readStringU8 expected error on short payload")
	}
}

// -------readBytesU16----------------------------------------------------------

func TestReadBytesU16_Empty(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	_, _ = buf.Write(u16BE(0))
	b, err := readBytesU16(buf)
	if err != nil {
		t.Fatalf("readBytesU16 error: %v", err)
	}
	if b != nil {
		t.Fatalf("readBytesU16 got %v, want nil", b)
	}
}

func TestReadBytesU16_Success(t *testing.T) {
	want := []byte{1, 2, 3, 4, 5}
	buf := bytes.NewBuffer(nil)
	_, _ = buf.Write(u16BE(uint16(len(want))))
	_, _ = buf.Write(want)
	got, err := readBytesU16(buf)
	if err != nil {
		t.Fatalf("readBytesU16 error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("readBytesU16 got %v, want %v", got, want)
	}
}

func TestReadBytesU16_ShortPayload(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	_, _ = buf.Write(u16BE(10)) // declare 10
	_, _ = buf.Write([]byte{1, 2, 3})
	_, err := readBytesU16(buf)
	if err == nil {
		t.Fatalf("readBytesU16 expected error on short payload")
	}
}

// -------readU8Len-------------------------------------------------------------

func TestReadU8Len_Success(t *testing.T) {
	buf := bytes.NewBuffer([]byte{200})
	n, err := readU8Len(buf)
	if err != nil {
		t.Fatalf("readU8Len error: %v", err)
	}
	if n != 200 {
		t.Fatalf("readU8Len got %d, want %d", n, 200)
	}
}

func TestReadU8Len_MaxBoundary(t *testing.T) {
	buf := bytes.NewBuffer([]byte{255})
	n, err := readU8Len(buf)
	if err != nil {
		t.Fatalf("readU8Len error: %v", err)
	}
	if n != 255 {
		t.Fatalf("readU8Len got %d, want 255", n)
	}
}

func TestReadU8Len_EOF(t *testing.T) {
	_, err := readU8Len(bytes.NewReader(nil))
	if err == nil {
		t.Fatalf("readU8Len expected error, got nil")
	}
}

// -------readU16Len------------------------------------------------------------

func TestReadU16Len_Success(t *testing.T) {
	buf := bytes.NewBuffer(u16BE(0x0123))
	n, err := readU16Len(buf)
	if err != nil {
		t.Fatalf("readU16Len error: %v", err)
	}
	if n != 0x0123 {
		t.Fatalf("readU16Len got %d, want %d", n, 0x0123)
	}
}

func TestReadU16Len_MaxBoundary(t *testing.T) {
	// Limit in code is 64KB-1 (65535), which equals max uint16.
	buf := bytes.NewBuffer(u16BE(65535))
	n, err := readU16Len(buf)
	if err != nil {
		t.Fatalf("readU16Len error: %v", err)
	}
	if n != 65535 {
		t.Fatalf("readU16Len got %d, want 65535", n)
	}
}

func TestReadU16Len_EOF(t *testing.T) {
	// Provide only one byte to force io.ErrUnexpectedEOF via binary.Read
	buf := bytes.NewBuffer([]byte{0x01})
	_, err := readU16Len(buf)
	if err == nil {
		t.Fatalf("readU16Len expected error, got nil")
	}
}

// -------sanity: integration style EOFs----------------------------------------

func TestReadFunctions_ReturnWrappedErrors(t *testing.T) {
	// readStringU8 should surface its wrapped error message when length prefix
	// exists but no body
	buf := bytes.NewBuffer([]byte{2}) // u8 len = 2, but no bytes follow
	_, err := readStringU8(buf)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	// We don't require exact string match; just ensure it's not io.EOF naked
	if err == io.EOF {
		t.Fatalf("expected wrapped error, got io.EOF")
	}
}
