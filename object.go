package rhizome

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// -----------------------------------------------------------------------------
// The primary parsing entry point.
// The main protocol version detection and parsing version handling.
// -----------------------------------------------------------------------------

// Response represents the ack value and the corresponding message's UID to
// send back to the producer.
type Response struct {
	Ack uint8
	UID string
}

// Object is a struct that is decoded from the incoming byte stream and ran
// through the system.
//
// Objects contain a *ConnResponder for communicating with the sender and a
// *Response for encoding a response with ack status and corresponding UID to
// send to the sender via the object.Responder.
type Object struct {
	// Which protocol decoding method that should be used to construct the
	// Object.
	Version uint8

	Responder *ConnResponder
	Response  *Response

	// ObjType is an application construct, it isn't concretely defined at the
	// protocol level.
	// ObjType is used to signify application domains the message object
	// pertains to.
	ObjType uint8

	// CmdType is an application construct, it isn't concretely defined at the
	// protocol level.
	// CmdType is used to signify what functionality the message object is
	// intended to invoke.
	CmdType uint8

	// The acknowledgement/negative acknowledgement status, like http return
	// code, signifying the result handling the Object.
	AckPlcy uint8

	// UID is an application construct, it isn't concretely defined at the
	// protocol level.
	// UID Is used for tracking purposes at the application level.
	UID string

	// Rhizome Objects contain 4 arguments in string form.
	// Applications may cast them to different types after decoding.
	// These are separate from the payload and are used to instruct the CmdType
	// in its execution.
	Arg1, Arg2 string
	Arg3, Arg4 string

	// What method of encoding to handle the payload bytes with.
	// The currently denoted
	PayloadEncoding PayloadEncoding

	// The generic information, if any, to forward to the subscribing system.
	Payload []byte
}

func NewObject(
	objType, cmdType, AckPlcy uint8,
	uid, arg1, arg2, arg3, arg4 string,
	payloadEncoding PayloadEncoding,
	payload []byte) *Object {

	return &Object{
		Version: ProtocolV1,

		Response: &Response{
			UID: uid,
			Ack: AckUnknown,
		},

		ObjType: objType,
		CmdType: cmdType,

		AckPlcy: AckPlcy,

		UID: uid,

		Arg1: arg1,
		Arg2: arg2,
		Arg3: arg3,
		Arg4: arg4,

		PayloadEncoding: payloadEncoding,
		Payload:         payload,
	}
}

// PrintValues prints each field on the object...
func (obj *Object) PrintValues() {
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("Obj Type:", obj.ObjType)
	fmt.Println("Cmd Type:", obj.CmdType)
	fmt.Println()

	if obj.Responder != nil {
		fmt.Println("Return Address:", obj.Responder.RemoteAddr())
	} else {
		fmt.Println("Return Address: nil")
	}
	fmt.Println()

	fmt.Println("UID:", obj.UID)
	fmt.Println()

	fmt.Println("Arg1:", obj.Arg1)
	fmt.Println("Arg2:", obj.Arg2)
	fmt.Println("Arg3:", obj.Arg3)
	fmt.Println("Arg4:", obj.Arg4)
	fmt.Println()

	fmt.Println("Payload Encoding:", obj.PayloadEncoding.String())
	fmt.Println()

	fmt.Println("Raw Payload:", string(obj.Payload))
	fmt.Println(strings.Repeat("-", 80))
}

// RespondWithAck sends the ack value (uint8) to the object's responder address.
// The ack value is application specific.
// Unlike other protocols, like HTTP, Rhizome does not have universal response
// codes.
//
// Responses are implemented at the application level.
// A broker may have a set of values, and some of those may overlap with another
// application like a database.
//
// Applications should have their own response APIs or built-in parsing or
// conversion functionality to make sense of application-specific acks/nacks.
func (obj *Object) RespondWithAck(ack uint8) error {
	if obj.Responder != nil {
		obj.Response.Ack = ack

		msg, err := obj.EncodeResponse()
		if err != nil {
			return err
		}

		return obj.Responder.Write(msg)
	}

	return errors.New("responder is nil")
}

// EncodeResponse serializes obj and returns an encoded byte array or error.
func (obj *Object) EncodeResponse() ([]byte, error) {
	switch obj.Version {

	case ProtocolV1:
		return EncodeResponseV1(*obj.Response), nil

	default:
		err := fmt.Errorf(
			"unable to encode response for %s", obj.Responder.RemoteAddr(),
		)
		return nil, err
	}
}

// parseProtoVer extracts only the protocol version and returns it along with
// a slice that starts at the next byte (i.e. the remainder of the message).
func parseProtoVer(data []byte) (uint8, []byte, error) {
	const u8len = 1
	if len(data) < u8len {
		return 0, nil, io.ErrUnexpectedEOF
	}
	ver := data[0]
	return ver, data[u8len:], nil
}

// DecodeFrame takes an array of bytes and a *ConnResponder to construct a
// *Object or error.
func DecodeFrame(line []byte, resp *ConnResponder) (*Object, error) {
	version, rest, err := parseProtoVer(line)
	if err != nil {
		err := fmt.Errorf("read protocol version: %v", err)
		return nil, err
	}

	obj := &Object{
		Version:   version,
		Responder: resp,
	}

	// Signal Weave apps always work off of the same type of object.
	// Message objects may evolve over time, adding new fields for new
	// functionality, but the application should remain compatible with previous
	// client side API versions.
	//
	// If a client is using API ver 1 to communicate with the application ver 2,
	// then the client should still be able to communicate.
	// This first token of a message is the API version, and this switch runs
	// the corresponding parsing logic.
	//
	// This is mainly because early on there was uncertainty if the protocol and
	// object structure were done right, and we reserved the ability to update
	// it as we go.
	switch version {

	case 1:
		return decodeV1(rest, obj)

	default:
		return nil, fmt.Errorf("unsupported protocol version: %d", obj.Version)
	}
}

// EncodeFrame serializes an Object into a single byte slice suitable for sending
// over the wire. It switches on obj.Version to remain forward-compatible.
func EncodeFrame(obj *Object) ([]byte, error) {
	switch obj.Version {
	case ProtocolV1:
		return encodeV1(obj)
	default:
		return nil, fmt.Errorf("unsupported protocol version: %d", obj.Version)
	}
}

// EncodeResponse creates ack/nack from object tokens or error.
func EncodeResponse(obj *Object) ([]byte, error) {
	switch obj.Version {

	case uint8(1):
		return EncodeResponseV1(*obj.Response), nil

	default:
		return nil, fmt.Errorf("unable to encode response for %s", obj.Responder.RemoteAddr())
	}
}
