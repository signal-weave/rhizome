# rhizome
The Rhizome protocol for use with Signal Weave applications.

Signal Weave applications share the Rhizome message object shape but treat
values differently depending on the application needs.

Objects will always be the same shape but unlike HTTP, a Rhizome object's field
values are not universal, instead field values are implemented at the
application and application-api level.

The first field in the protocol is the version number which dictates the
decoding method to generate a `Rhizome.Object`.

Rhizome message objects look like the following:

```go
type Object struct {
	// Which protocol decoding method that shoud be used to construct the
	// Object.
	Protocol uint8

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
	// UID Is used for tracking purposes at the applicatoin level.
	UID string

	// Rhizome Objects contain 4 arguments in string form.
	// Applicatoins may cast them to different types after decoding.
	// These are separate from the payload, and are uesd to instruct the CmdType
	// in its execution.
	Arg1, Arg2 string
	Arg3, Arg4 string

	// The generic information, if any, to forward to the subscribing system.
	Payload []byte
}
```
