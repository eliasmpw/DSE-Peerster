package gossiper

import (
	"github.com/dedis/protobuf"
	"net"
)

func onSimpleMessageReceived(gsspr *Gossiper, message []byte,	sourceAddr *net.UDPAddr, isClient bool) {
	var packetReceived = GossipPacket{}
	protobuf.Decode(message, &packetReceived)
	if packetReceived.Simple == nil {
		return
	}
	if isClient {
		//Log the line for client message
		logClientMessage(packetReceived)
		for _, peer := range gsspr.peersList {
			packetReceived.Simple.OriginalName = gsspr.Name
			packetReceived.Simple.RelayPeerAddr = gsspr.addressStr
			if peer != gsspr.addressStr {
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet: packetReceived,
					destination: peer,
				}
			}
		}
	} else {
		//Log the first line for simple message
		logSimpleMessage(packetReceived, sourceAddr.String())
		packetReceived.Simple.RelayPeerAddr = gsspr.addressStr
		found := false
		for _, peer := range gsspr.peersList {
			if peer == sourceAddr.String() {
				found = true
			} else {
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet: packetReceived,
					destination: peer,
				}
			}
		}
		if !found {
			gsspr.peersList = append(gsspr.peersList, sourceAddr.String())
		}
		logPeers(gsspr)
	}
}
