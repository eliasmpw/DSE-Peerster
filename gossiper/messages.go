package gossiper

// Message types
type SimpleMessage struct {
	OriginalName string
	RelayPeerAddr string
	Contents string
}

type RumorMessage struct {
	Origin string
	ID uint32
	Text string
}

type PeerStatus struct {
	Identifier string
	NextID uint32
}

type PrivateMessage struct {
	Origin string
	ID uint32
	Text string
	Destination string
	HopLimit uint32
}

// Gossip packet
type GossipPacket struct {
	Simple *SimpleMessage
	Rumor *RumorMessage
	Status *StatusPacket
	Private *PrivateMessage
}

// QueuedMessage
type QueuedMessage struct {
	packet GossipPacket
	destination string
}
