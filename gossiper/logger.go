package gossiper

import (
	"fmt"
	"strings"
)

func logClientMessage(packetReceived GossipPacket) {
	fmt.Printf("CLIENT MESSAGE %s\n", packetReceived.Simple.Contents)
}

func logSimpleMessage(packetReceived GossipPacket, sourceAddr string) {
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n",
		packetReceived.Simple.OriginalName,
		sourceAddr,
		packetReceived.Simple.Contents)
}

func logPeers(gsspr *Gossiper) {
	peersString := []string{}
	for i := range gsspr.peersList {
		peer := gsspr.peersList[i]
		peersString = append(peersString, peer)
	}
	// Join our string slice.
	result := strings.Join(peersString, ",")
	//Log the peer list line
	fmt.Printf("PEERS %s\n", result)
}

func logMongering(destPeer string) {
	fmt.Printf("MONGERING with %s\n", destPeer)
}

func LogFlippedCoin(destPeer string) {
	fmt.Printf("FLIPPED COIN sending rumor to %s\n", destPeer)
}

func logRumorMessage(packetReceived GossipPacket, relayAddr string) {
	fmt.Printf("RUMOR origin %s from %s ID %d contents %s\n",
		packetReceived.Rumor.Origin, relayAddr, packetReceived.Rumor.ID, packetReceived.Rumor.Text)
}

func logStatusMessage(packetReceived GossipPacket, relayAddr string) {
	logStr := "STATUS from " + relayAddr
	for _, status := range packetReceived.Status.Want {
		logStr += fmt.Sprintf(" peer %s nextID %d", status.Identifier, status.NextID)
	}
	fmt.Printf(logStr + "\n")
}

func LogSync(addr string) {
	fmt.Printf("IN SYNC WITH %s\n", addr)
}

func logAntiEntropy(peerAddr string) {
	fmt.Printf("ANTIENTROPY TO %s\n", peerAddr)
}

func logRoutingTableUpdate(peerName string, peerAddr string) {
	fmt.Printf("DSDV %s %s\n", peerName, peerAddr)
}

func logPrivateMessage(packetReceived GossipPacket) {
	fmt.Printf("PRIVATE origin %s hop-limit %d contents %s\n",
		packetReceived.Private.Origin, packetReceived.Private.HopLimit,
		packetReceived.Private.Text)
}
