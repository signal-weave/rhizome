package rhizome

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

//------------------------------------------------------------------------------
// Byte buffer handling for decoding string, byte arrays, and unisnged integer
// values.

// Notably these do not create Mycelia error types because they were very low
// level and the callers give more context creating them instead.
//------------------------------------------------------------------------------

//--------Integers--------------------------------------------------------------

func readU8(r io.Reader, out *uint8) error {
	// endian is irrelevant for 1 byte but Read() requires it.
	return binary.Read(r, binary.BigEndian, out)
}

//--------Strings---------------------------------------------------------------

// Read string up to 65535 characters long.
func readStringU8(r io.Reader) (string, error) {
	n, err := readU8Len(r)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("read string bytes: %w", err)
	}
	return string(buf), nil
}

//--------Bytes-----------------------------------------------------------------

// Read bytes up to 65535 bytes long.
func readBytesU16(r io.Reader) ([]byte, error) {
	n, err := readU16Len(r)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read payload bytes: %w", err)
	}
	return buf, nil
}

//--------Field Prefixes--------------------------------------------------------

// Read from the io.Reader up to 255 bytes forwards.
func readU8Len(r io.Reader) (uint8, error) {
	var n uint8
	var u8Limit uint8 = 255

	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return 0, fmt.Errorf("read length: %w", err)
	}
	// Sanity check - May want to store this value somewhere.
	if n > u8Limit {
		return 0, errors.New("declared length exceeds 255byte safety limit")
	}
	return n, nil
}

// Read from the io.Reader up to 65535 bytes forwards.
func readU16Len(r io.Reader) (uint16, error) {
	var n uint16
	var u16Limit uint16 = 64*BytesInKilobyte - 1 // 64KB - 1B

	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return 0, fmt.Errorf("read length: %w", err)
	}
	// Sanity check - May want to store this value somewhere.
	if n > u16Limit {
		return 0, errors.New("declared length exceeds 64KB safety limit")
	}
	return n, nil
}
