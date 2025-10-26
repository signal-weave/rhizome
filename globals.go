package rhizome

// -------Size------------------------------------------------------------------

const (
	BytesInKilobyte = 1024
)

// -------Objects---------------------------------------------------------------

const (
	ProtocolV1 = uint8(1)
)

const (
	ObjUnknown uint8 = 0

	ObjDelivery    uint8 = 1
	ObjTransformer uint8 = 2
	ObjSubscriber  uint8 = 3
	ObjChannel     uint8 = 4

	ObjGlobals uint8 = 20

	ObjAction uint8 = 50
)

const (
	CmdUnknown uint8 = 0

	CmdSend   uint8 = 1
	CmdAdd    uint8 = 2
	CmdRemove uint8 = 3

	CmdUpdate uint8 = 20

	CmdSigterm uint8 = 50
)

// -------Acks/Nacks------------------------------------------------------------

const (
	// AckPlcyNoreply signifies sender does not wish to receive ack.
	AckPlcyNoreply uint8 = 0

	// AckPlcyOnsent signifies sender wants to get ack when broker delivers to
	// final subscriber.
	// This often means sending the ack back after the final channel has
	// processed the message object.
	AckPlcyOnsent uint8 = 1
)

const (
	AckUnknown uint8 = 0 // Undetermined

	// AckSent means broker was able to and finished sending message to
	// subscribers.
	AckSent uint8 = 1

	// AckTimeout is generated as a returned value if no ack aws gotten before
	// the timeout time elapsed.
	AckTimeout uint8 = 10

	AckChannelNotFound      uint8 = 20
	AckChannelAlreadyExists uint8 = 21
	AckRouteNotFound        uint8 = 30
)
