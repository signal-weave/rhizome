package rhizome

import (
	"bytes"
	"testing"
)

func TestEncodeResponse_V1_MatchesEncodeResponseV1(t *testing.T) {
	// Set up a responder with a fake remote address.
	fc := newFakeConn("10.0.0.9:12345")
	cr := &ConnResponder{C: fc}

	// Build a minimal Response and Object for protocol v1.
	resp := Response{
		UID: "abc-123",
		Ack: ACK_SENT,
	}
	obj := &Object{
		Protocol:  PROTOCOL_V1,
		Response:  &resp,
		Responder: cr,
	}

	// Act
	got, err := EncodeResponse(obj)
	if err != nil {
		t.Fatalf("EncodeResponse returned error: %v", err)
	}

	// Expect exact bytes as produced by the v1 encoder.
	want := EncodeResponseV1(resp)
	if !bytes.Equal(got, want) {
		t.Fatalf(
			"EncodeResponse(v1) bytes mismatch.\n got=%v\nwant=%v", got, want,
		)
	}
}

func TestEncodeResponse_UnsupportedProtocol_ReturnsErrorWithRemoteAddr(
	t *testing.T,
) {
	fc := newFakeConn("192.168.1.77:6000")
	cr := &ConnResponder{C: fc}

	// Use an unsupported protocol (anything not PROTOCOL_V1).
	obj := &Object{
		Protocol:  0,
		Response:  nil, // not used along this branch
		Responder: cr,
	}

	out, err := EncodeResponse(obj)
	if err == nil {
		t.Fatalf("expected error for unsupported protocol, got nil")
	}
	if out != nil {
		t.Fatalf("expected nil output on error, got %v", out)
	}
	// Error should mention the remote address.
	if !bytes.Contains([]byte(err.Error()), []byte("192.168.1.77:6000")) {
		t.Fatalf("error should include remote addr; got: %v", err)
	}
}
