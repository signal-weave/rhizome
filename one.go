package rhizome

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// -----------------------------------------------------------------------------
// Version 1 object decoding.
// -----------------------------------------------------------------------------
// *Note that this is a messaging protocol, not a file transfer protocol.
// -----------------------------------------------------------------------------
// The version 1 protocol looks as follows:

// # Fixed field sized header
// +---------+--------+-------------+-------------+---------------+
// | u32 len | u8 ver | u8 obj_type | u8 cmd_type | u8 ack policy |
// +---------+--------+-------------+-------------+---------------+

// which is then followed by a variable field sized sub-header that contains a
// UID and the sender's address for tracking purposes.

// # Tracking Sub-header
// +-------------+---------------------+
// | u8 len uid  | u16 len return addr |
// +-------------+---------------------+

// which is then followed by 4 uint8 sized byte fields that act as arguments for
// the object type in the fixed header.
// Because these are byte streams, all arguments are considered string types
// unless the executor casts them to another type.

// # Argument Sub-Header
// +---------------+---------------+---------------+---------------+
// |  u8 len arg1  |  u8 len arg2  |  u8 len arg3  |  u8 len arg4  |
// +---------------+---------------+---------------+---------------+

// And finally, the message payload that would be delivered to external sources
// and the encoding method used to pack it into the message. Consumers can
// reference this field for decoding.
// If this is unused because the message is changing the internals of the broker
// at runtime, then the field defaults to a value of 0x00.

// # Globals Body
// +------------------+-----------------+
// | u8 encoding type | u16 len payload |
// +------------------+-----------------+

// -----------------------------------------------------------------------------
// Responses are a three-field message: message length prefix, corresponding
// uid, and the ack/nack value.

// +---------+------------+--------------+
// | u16 len | u8 len uid | u8 ack value |
// +---------+------------+--------------+

// -----------------------------------------------------------------------------

//--------Decoding--------------------------------------------------------------

func decodeV1(data []byte, obj *Object) (*Object, error) {
	r := bytes.NewReader(data)

	// ObjType + CmdType
	obj, err := parseBaseHeader(r, obj)
	if err != nil {
		return nil, err
	}
	// UID + Source Address
	obj, err = parseTrackingHeader(r, obj)
	if err != nil {
		return nil, err
	}
	// Arg fields
	obj, err = parseArgumentFields(r, obj)
	if err != nil {
		return nil, err
	}
	// PayloadEncoding
	obj, err = parsePayloadEncoding(r, obj)
	if err != nil {
		return nil, err
	}
	// Payload
	payload, err := readBytesU16(r)
	if err != nil {
		msg := fmt.Sprintf("Unable to parse payload from %s: %s", obj.Responder.RemoteAddr(), err)
		err := errors.New(msg)
		return nil, err
	}
	obj.Payload = payload

	// Response
	response := &Response{
		UID: obj.UID,
		Ack: AckUnknown,
	}
	obj.Response = response

	if r.Len() != 0 {
		obj = nil
		err = errors.New("unaccounted data in reader")
	}

	return obj, err
}

// Parses the header after version: obj_type, and cmd_type from message.
func parseBaseHeader(r io.Reader, cmd *Object) (*Object, error) {
	if err := readU8(r, &cmd.ObjType); err != nil {
		err := fmt.Errorf(
			"unable to parse u8 ObjType field from message: %s", err,
		)
		return nil, err
	}

	if err := readU8(r, &cmd.CmdType); err != nil {
		err := fmt.Errorf(
			"unable to parse u8 CmdType field from message: %s", err,
		)
		return nil, err
	}

	if err := readU8(r, &cmd.AckPlcy); err != nil {
		err := fmt.Errorf(
			"unable to parse u8 AckPolicy field from message: %s", err,
		)
		return nil, err
	}

	return cmd, nil
}

// Parses the UID and sender address from the reader.
func parseTrackingHeader(r io.Reader, cmd *Object) (*Object, error) {
	// UID field comes before sender address field.
	uid, err := readStringU8(r)
	if err != nil {
		err := fmt.Errorf(
			"unable to parse string UID field from message: %s", err,
		)
		return nil, err
	}
	if uid == "" {
		err := fmt.Errorf(
			"empty UID field from mesage: %s",
			cmd.Responder.C.LocalAddr().String(),
		)
		return nil, err
	}
	cmd.UID = uid

	return cmd, nil
}

func parseArgError(pos int, fromAddr string, err error) error {
	return fmt.Errorf(
		"unable to parse argument position %d for %s: %s",
		pos, fromAddr, err,
	)
}

// Parse the four argument fields from the reader.
func parseArgumentFields(r io.Reader, cmd *Object) (*Object, error) {
	arg1, err := readStringU8(r)
	fromAddr := cmd.Responder.RemoteAddr()
	if err != nil {
		return nil, parseArgError(1, fromAddr, err)
	}
	cmd.Arg1 = arg1

	arg2, err := readStringU8(r)
	if err != nil {
		return nil, parseArgError(2, fromAddr, err)
	}
	cmd.Arg2 = arg2

	arg3, err := readStringU8(r)
	if err != nil {
		return nil, parseArgError(3, fromAddr, err)
	}
	cmd.Arg3 = arg3

	arg4, err := readStringU8(r)
	if err != nil {
		return nil, parseArgError(4, fromAddr, err)
	}
	cmd.Arg4 = arg4

	return cmd, nil
}

func parsePayloadEncoding(r io.Reader, cmd *Object) (*Object, error) {
	if err := readU8(r, (*uint8)(&cmd.PayloadEncoding)); err != nil {
		err := fmt.Errorf("unable to parse payload encoding type from %s", cmd.Responder.RemoteAddr())
		return nil, err
	}
	return cmd, nil
}

//--------Encoding--------------------------------------------------------------

// encodeV1 builds a v1 message:
//
// [ u8 ver ][ u8 obj_type ][ u8 cmd_type ][ u8 ack_policy ]
// [ u8 len uid ][ u8 len arg1 ][ u8 len arg2 ][ u8 len arg3 ][ u8 len arg4 ]
// [ u8 encoding ][ u16 len payload ][ payload... ]
func encodeV1(obj *Object) ([]byte, error) {
	// Basic validation to match decoder expectations.
	if obj.UID == "" {
		return nil, errors.New("encodeV1: UID must not be empty")
	}
	if len(obj.Payload) > 64*BytesInKilobyte-1 {
		return nil, fmt.Errorf("encodeV1: payload too large: %d bytes", len(obj.Payload))
	}

	body := bytes.NewBuffer(nil)

	// Version
	writeU8(body, ProtocolV1)

	// Fixed header
	writeU8(body, obj.ObjType)
	writeU8(body, obj.CmdType)
	writeU8(body, obj.AckPlcy)

	// Tracking + arguments (all u8-len strings)
	if err := writeString8(body, obj.UID); err != nil {
		return nil, fmt.Errorf("encodeV1: uid: %w", err)
	}
	if err := writeString8(body, obj.Arg1); err != nil {
		return nil, fmt.Errorf("encodeV1: arg1: %w", err)
	}
	if err := writeString8(body, obj.Arg2); err != nil {
		return nil, fmt.Errorf("encodeV1: arg2: %w", err)
	}
	if err := writeString8(body, obj.Arg3); err != nil {
		return nil, fmt.Errorf("encodeV1: arg3: %w", err)
	}
	if err := writeString8(body, obj.Arg4); err != nil {
		return nil, fmt.Errorf("encodeV1: arg4: %w", err)
	}

	// Payload encoding (u8) + payload (u16-len + bytes)
	writeU8(body, uint8(obj.PayloadEncoding))
	writeU16(body, uint16(len(obj.Payload)))
	if len(obj.Payload) != 0 {
		body.Write(obj.Payload)
	}

	return body.Bytes(), nil
}

//--------Response--------------------------------------------------------------

// EncodeResponseV1 encodes a protocol.Response object into []byte.
func EncodeResponseV1(response Response) []byte {
	body := bytes.NewBuffer(nil)
	_ = writeString8(body, response.UID)
	writeU8(body, response.Ack)

	full := bytes.NewBuffer(nil)
	writeU16(full, uint16(body.Len()))
	full.Write(body.Bytes())
	return full.Bytes()
}
