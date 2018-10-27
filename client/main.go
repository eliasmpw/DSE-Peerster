package main

import (
	"flag"
	"github.com/dedis/protobuf"
	"github.com/eliasmpw/Peerster/common"
	"github.com/eliasmpw/Peerster/gossiper"
	"net"
)

func main() {
	// Load values passed via flags
	uiPort := flag.String("UIPort", "8080", "Port for the UI client");
	msg := flag.String("msg", "", "Message to be sent");
	flag.Parse()

	messageToSend := gossiper.SimpleMessage{
		OriginalName: "",
		RelayPeerAddr: "",
		Contents: *msg,
	}
	packetToSend := gossiper.GossipPacket{
		Simple: &messageToSend,
	}

	// Send packet
	content, err := protobuf.Encode(&packetToSend)
	common.CheckError(err)
	addressToSend, err := net.ResolveUDPAddr("udp4", "127.0.0.1:" + *uiPort)
	common.CheckError(err)
	udpConnection, err := net.DialUDP("udp4", nil, addressToSend)
	common.CheckError(err)
	udpConnection.Write(content)
	udpConnection.Close()
}
