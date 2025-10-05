package rhizome

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// -------writeU8---------------------------------------------------------------

func TestWriteU8_WritesSingleByte(t *testing.T) {
	var buf bytes.Buffer
	writeU8(&buf, 0x7F)

	got := buf.Bytes()
	if len(got) != 1 {
		t.Fatalf("writeU8 wrote %d bytes, want 1", len(got))
	}
	if got[0] != 0x7F {
		t.Fatalf("writeU8 wrote 0x%02X, want 0x7F", got[0])
	}
}

// -------writeU16--------------------------------------------------------------

func TestWriteU16_WritesBigEndian(t *testing.T) {
	var buf bytes.Buffer
	writeU16(&buf, 0x0123)

	got := buf.Bytes()
	if len(got) != 2 {
		t.Fatalf("writeU16 wrote %d bytes, want 2", len(got))
	}
	if got[0] != 0x01 || got[1] != 0x23 {
		t.Fatalf("writeU16 wrote %v, want [0x01 0x23]", got)
	}
}

// -------writeString8----------------------------------------------------------

func TestWriteString8_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := writeString8(&buf, ""); err != nil {
		t.Fatalf("writeString8 error: %v", err)
	}
	got := buf.Bytes()
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("writeString8('') wrote %v, want [0x00]", got)
	}
}

func TestWriteString8_SimpleASCII(t *testing.T) {
	var buf bytes.Buffer
	if err := writeString8(&buf, "abc"); err != nil {
		t.Fatalf("writeString8 error: %v", err)
	}
	got := buf.Bytes()
	want := append([]byte{3}, []byte("abc")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("writeString8('abc') wrote %v, want %v", got, want)
	}
}

func TestWriteString8_TooLong(t *testing.T) {
	// length prefix is u8, so >255 should fail
	var buf bytes.Buffer
	long := strings.Repeat("x", 256)
	if err := writeString8(&buf, long); err == nil {
		t.Fatalf("writeString8(len=256) expected error, got nil")
	}
}

// -------WriteU16Len-----------------------------------------------------------

func TestWriteU16Len_WritesTwoBytesAndMatchesValue(t *testing.T) {
	var buf bytes.Buffer

	// We expect exactly two bytes, big-endian, equal to uint16(n).
	// If the current implementation panics (e.g., by calling PutUint32 into a 2-byte slice),
	// this test will fail with a clear message.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WriteU16Len panicked: %v (expected to write 2 bytes using PutUint16)", r)
		}
	}()

	WriteU16Len(&buf, 0xBEEF)

	got := buf.Bytes()
	if len(got) != 2 {
		t.Fatalf("WriteU16Len wrote %d bytes, want 2", len(got))
	}
	var n uint16
	_ = binary.Read(bytes.NewReader(got), binary.BigEndian, &n)
	if n != 0xBEEF {
		t.Fatalf("WriteU16Len encoded 0x%04X, want 0xBEEF", n)
	}
}
