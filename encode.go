package rhizome

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

//--------Integers--------------------------------------------------------------

// writeU8 converts uint8 value n into byte, inserting it into buf.
func writeU8(buf *bytes.Buffer, n uint8) {
	_ = buf.WriteByte(n)
}

// writeU16 converts uint16 value n into bytes, inserting it into buf.
func writeU16(buf *bytes.Buffer, n uint16) {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], n)
	buf.Write(tmp[:])
}

//--------String----------------------------------------------------------------

// writeString8 converts uint8 len string s into a byte array.
func writeString8(buf *bytes.Buffer, s string) error {
	b := []byte(s)
	if len(b) > 255 {
		return fmt.Errorf("string too long for u8 prefix: %d", len(b))
	}
	writeU8(buf, uint8(len(b)))
	buf.Write(b)
	return nil
}

//--------Field Prefixes--------------------------------------------------------

// WriteU16Len prefixes with total length (u16 big-endian).
func WriteU16Len(buf *bytes.Buffer, n uint16) {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], n)
	buf.Write(tmp[:])
}
