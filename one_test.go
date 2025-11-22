package rhizome

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// -------net.Conn Substitute---------------------------------------------------

type dummyAddr struct{ s string }

func (d dummyAddr) Network() string { return "dummy" }
func (d dummyAddr) String() string  { return d.s }

type dummyConn struct{}

func (d *dummyConn) Read(b []byte) (int, error)         { return 0, io.EOF }
func (d *dummyConn) Write(b []byte) (int, error)        { return len(b), nil }
func (d *dummyConn) Close() error                       { return nil }
func (d *dummyConn) LocalAddr() net.Addr                { return dummyAddr{"local:0"} }
func (d *dummyConn) RemoteAddr() net.Addr               { return dummyAddr{"remote:0"} }
func (d *dummyConn) SetDeadline(t time.Time) error      { return nil }
func (d *dummyConn) SetReadDeadline(t time.Time) error  { return nil }
func (d *dummyConn) SetWriteDeadline(t time.Time) error { return nil }

func newResponder() *ConnResponder {
	return NewConnResponder(&dummyConn{})
}

// -------Helpers---------------------------------------------------------------

func assertObjectsEqual(t *testing.T, want, got *Object) {
	t.Helper()

	if got.Version != want.Version {
		t.Fatalf("Version mismatch: got %d want %d", got.Version, want.Version)
	}
	if got.ObjType != want.ObjType {
		t.Fatalf("ObjType mismatch: got %d want %d", got.ObjType, want.ObjType)
	}
	if got.CmdType != want.CmdType {
		t.Fatalf("CmdType mismatch: got %d want %d", got.CmdType, want.CmdType)
	}
	if got.AckPlcy != want.AckPlcy {
		t.Fatalf("AckPlcy mismatch: got %d want %d", got.AckPlcy, want.AckPlcy)
	}
	if got.UID != want.UID {
		t.Fatalf("UID mismatch: got %q want %q", got.UID, want.UID)
	}
	if got.Arg1 != want.Arg1 || got.Arg2 != want.Arg2 || got.Arg3 != want.Arg3 || got.Arg4 != want.Arg4 {
		t.Fatalf("arg mismatch: got (%q,%q,%q,%q) want (%q,%q,%q,%q)",
			got.Arg1, got.Arg2, got.Arg3, got.Arg4, want.Arg1, want.Arg2, want.Arg3, want.Arg4)
	}
	if got.PayloadEncoding != want.PayloadEncoding {
		t.Fatalf("PayloadEncoding mismatch: got %v want %v", got.PayloadEncoding, want.PayloadEncoding)
	}
	if !bytes.Equal(got.Payload, want.Payload) {
		t.Fatalf("Payload mismatch: got %v want %v", got.Payload, want.Payload)
	}
}

// -------Encoding / Decoding---------------------------------------------------

func TestEncodeV1_RoundTrip_Basic(t *testing.T) {
	obj := NewObject(
		ObjDelivery, CmdSend, AckUnknown,
		"uid-123", "arg1", "arg2", "arg3", "arg4",
		EncodingJson,
		[]byte(`{"hello":"world"}`),
	)
	obj.Version = ProtocolV1

	encoded, err := encodeV1(obj)
	if err != nil {
		t.Fatalf("encodeV1 error: %v", err)
	}

	// version byte should be first
	if len(encoded) == 0 || encoded[0] != ProtocolV1 {
		t.Fatalf("expected first byte to be ProtocolV1 (%d), got %v", ProtocolV1, encoded[:1])
	}

	// Decode the whole frame (DecodeFrame expects the version byte to be present)
	round, err := DecodeFrame(encoded, newResponder())
	if err != nil {
		t.Fatalf("DecodeFrame error: %v", err)
	}

	assertObjectsEqual(t, obj, round)
}

func TestEncodeV1_RoundTrip_EmptyPayload(t *testing.T) {
	obj := NewObject(
		ObjDelivery, CmdSend, AckUnknown,
		"uid-000", "", "", "", "",
		EncodingNA,
		nil,
	)
	obj.Version = ProtocolV1

	encoded, err := encodeV1(obj)
	if err != nil {
		t.Fatalf("encodeV1 error: %v", err)
	}

	// Decode
	round, err := DecodeFrame(encoded, newResponder())
	if err != nil {
		t.Fatalf("DecodeFrame error: %v", err)
	}

	assertObjectsEqual(t, obj, round)
}

func TestEncodeV1_PayloadIntegrity(t *testing.T) {
	payload := bytes.Repeat([]byte{0xAB}, 64) // arbitrary payload
	obj := NewObject(
		ObjDelivery, CmdSend, AckUnknown,
		"uid-payload", "a1", "a2", "a3", "a4",
		EncodingYaml,
		payload,
	)
	obj.Version = ProtocolV1

	encoded, err := encodeV1(obj)
	if err != nil {
		t.Fatalf("encodeV1 error: %v", err)
	}

	round, err := DecodeFrame(encoded, newResponder())
	if err != nil {
		t.Fatalf("DecodeFrame error: %v", err)
	}

	if !bytes.Equal(round.Payload, payload) {
		t.Fatalf("payload mismatch: got %v, want %v", round.Payload, payload)
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
