package rhizome

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
)

// -------Message---------------------------------------------------------------

// encodeV1MessageFrame constructs a complete v1 message frame buffer for testing.
func encodeV1MessageFrame(
	version uint8, objType, cmdType, ackPolicy uint8, uid string,
	args [4]string, encoding PayloadEncoding, payload []byte,
) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Version byte first
	writeU8(buf, version)

	// Base header
	writeU8(buf, objType)
	writeU8(buf, cmdType)
	writeU8(buf, ackPolicy)

	// UID (string u8)
	if err := writeString8(buf, uid); err != nil {
		return nil, err
	}

	// 4 argument strings
	for _, a := range args {
		if err := writeString8(buf, a); err != nil {
			return nil, err
		}
	}

	// Payload encoding (u8)
	writeU8(buf, uint8(encoding))

	// Payload (u16 length + data)
	writeU16(buf, uint16(len(payload)))
	buf.Write(payload)

	return buf.Bytes(), nil
}

func TestDecodeFrameMatches(t *testing.T) {
	args := [4]string{"arg1", "arg2", "", ""}
	payload := []byte("hello")

	msg, err := encodeV1MessageFrame(
		ProtocolV1,
		ObjDelivery,
		CmdSend,
		AckPlcyNoreply,
		"uid-123",
		args,
		EncodingJson,
		payload,
	)
	if err != nil {
		t.Fatalf("buildV1Message error: %v", err)
	}

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	resp := NewConnResponder(c1)

	obj, err := DecodeFrame(msg, resp)
	if err != nil {
		t.Fatalf("DecodeFrame error: %v", err)
	}

	if obj.Version != ProtocolV1 {
		t.Fatalf("Protocol Version mismatch: got %v, want %v", obj.Version, "1")
	}
	if obj.ObjType != ObjDelivery {
		t.Fatalf("Object type mismatch: got %v, want %v", obj.ObjType, ObjDelivery)
	}
	if obj.CmdType != CmdSend {
		t.Fatalf("Command type mismatch: got %v, want %v", obj.CmdType, CmdSend)
	}
	if obj.AckPlcy != AckPlcyNoreply {
		t.Fatalf("Ack policy mismatch: got %v, want %v", obj.AckPlcy, AckPlcyNoreply)
	}
	if obj.UID != "uid-123" {
		t.Fatalf("UID mismatch: got %q", obj.UID)
	}

	if obj.Arg1 != "arg1" {
		t.Fatalf("arg1 mismatch: got %s, want %s", obj.Arg1, "arg1")
	}
	if obj.Arg2 != "arg2" {
		t.Fatalf("arg1 mismatch: got %s, want %s", obj.Arg2, "arg2")
	}
	if obj.Arg3 != "" {
		t.Fatalf("arg1 mismatch: got %s, want %s", obj.Arg3, "")
	}
	if obj.Arg4 != "" {
		t.Fatalf("arg1 mismatch: got %s, want %s", obj.Arg4, "")
	}

	if obj.PayloadEncoding != EncodingJson {
		t.Fatalf("PayloadEncoding mismatch: got %v, want %v", obj.PayloadEncoding, EncodingJson)
	}
	if got, want := obj.PayloadEncoding.String(), "json"; got != want {
		t.Fatalf("PayloadEncoding.String mismatch: got %q, want %q", got, want)
	}
	if string(obj.Payload) != string(payload) {
		t.Fatalf("Payload mismatch: got %q, want %q", string(obj.Payload), string(payload))
	}

}

// -------Response--------------------------------------------------------------

func decodeV1ResponseFrame(t *testing.T, frame []byte) (uid string, ack byte) {
	t.Helper()

	// Read u16 length prefix (big-endian)
	if len(frame) < 2 {
		t.Fatalf("frame too short for u16 length: %v", frame)
	}
	n := binary.BigEndian.Uint16(frame[:2])
	body := frame[2:]

	if int(n) != len(body) {
		t.Fatalf("declared length %d does not match body len %d", n, len(body))
	}

	// Read u8 string length
	if len(body) < 1 {
		t.Fatalf("body too short to contain string length")
	}
	strLen := int(body[0])
	body = body[1:]

	if len(body) < strLen+1 { // +1 for ack byte
		t.Fatalf("body too short: need %d bytes for uid + 1 for ack, got %d", strLen+1, len(body))
	}

	uid = string(body[:strLen])
	ack = body[strLen]
	body = body[strLen+1:]

	if len(body) != 0 {
		t.Fatalf("trailing bytes after uid+ack: %v", body)
	}
	return uid, ack
}

func TestEncodeResponseV1_EmptyUID(t *testing.T) {
	resp := Response{
		UID: "",
		Ack: AckSent, // 1
	}

	got := EncodeResponseV1(resp)

	// Declared length should be 2 (u8 strlen=0 + u8 ack)
	if len(got) < 2 {
		t.Fatalf("frame too short: %v", got)
	}
	decl := binary.BigEndian.Uint16(got[:2])
	if decl != 2 {
		t.Fatalf("declared length = %d, want 2", decl)
	}

	uid, ack := decodeV1ResponseFrame(t, got)
	if uid != "" {
		t.Fatalf("uid = %q, want empty", uid)
	}
	if ack != AckSent {
		t.Fatalf("ack = %d, want %d", ack, AckSent)
	}
}

func TestEncodeResponseV1_ShortUID(t *testing.T) {
	resp := Response{
		UID: "abc",
		Ack: AckSent, // 1
	}

	got := EncodeResponseV1(resp)

	// Body is 1 (strlen) + 3 (abc) + 1 (ack) = 5
	if len(got) < 2 {
		t.Fatalf("frame too short: %v", got)
	}
	decl := binary.BigEndian.Uint16(got[:2])
	if decl != 5 {
		t.Fatalf("declared length = %d, want 5", decl)
	}

	uid, ack := decodeV1ResponseFrame(t, got)
	if uid != "abc" {
		t.Fatalf("uid = %q, want %q", uid, "abc")
	}
	if ack != AckSent {
		t.Fatalf("ack = %d, want %d", ack, AckSent)
	}
}

func TestEncodeResponseV1_MaxUIDLen255(t *testing.T) {
	uidBytes := bytes.Repeat([]byte{'x'}, 255)
	resp := Response{
		UID: string(uidBytes),
		Ack: 42, // arbitrary ack to ensure it lands after the string
	}

	got := EncodeResponseV1(resp)

	// Body is 1 (strlen) + 255 (uid) + 1 (ack) = 257
	if len(got) < 2 {
		t.Fatalf("frame too short: %v", got)
	}
	decl := binary.BigEndian.Uint16(got[:2])
	if decl != 257 {
		t.Fatalf("declared length = %d, want 257", decl)
	}

	uid, ack := decodeV1ResponseFrame(t, got)
	if uid != string(uidBytes) {
		t.Fatalf("uid mismatch: len(got)=%d, want=%d", len(uid), len(uidBytes))
	}
	if ack != 42 {
		t.Fatalf("ack = %d, want %d", ack, 42)
	}
}

func TestEncodeResponseV1_FrameIsExactlyPrefixPlusBody(t *testing.T) {
	resp := Response{UID: "z", Ack: 9}

	got := EncodeResponseV1(resp)

	// prefix (2) + body (1 strlen + 1 uid + 1 ack) = 5 total bytes
	if len(got) != 5 {
		t.Fatalf("frame total len = %d, want 5 bytes", len(got))
	}

	// Ensure we can fully read with io.ReadFull-style behavior
	r := bytes.NewReader(got)
	// Read prefix
	var plen uint16
	if err := binary.Read(r, binary.BigEndian, &plen); err != nil {
		t.Fatalf("binary.Read prefix: %v", err)
	}
	if plen != 3 {
		t.Fatalf("prefix length = %d, want 3", plen)
	}

	// Read body
	body := make([]byte, plen)
	if _, err := io.ReadFull(r, body); err != nil {
		t.Fatalf("io.ReadFull body: %v", err)
	}
	// No trailing bytes
	if r.Len() != 0 {
		t.Fatalf("expected no trailing bytes, have %d", r.Len())
	}

	// Body should be: [0x01 'z' 0x09]
	want := []byte{0x01, 'z', 0x09}
	if !bytes.Equal(body, want) {
		t.Fatalf("body = %v, want %v", body, want)
	}
}
