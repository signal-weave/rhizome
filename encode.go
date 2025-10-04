package rhizome

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

//--------Integers--------------------------------------------------------------

func writeU8(buf *bytes.Buffer, n uint8) {
	_ = buf.WriteByte(n)
}

func writeU16(buf *bytes.Buffer, n uint16) {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], n)
	buf.Write(tmp[:])
}

//--------String----------------------------------------------------------------

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

// Prefix with total length (u16 big-endian).
func WriteU16Len(buf *bytes.Buffer, n uint32) {
	var tmp [2]byte
	binary.BigEndian.PutUint32(tmp[:], n)
	buf.Write(tmp[:])
}
