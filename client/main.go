package main

import (
	"encoding/hex"
	"flag"
	"github.com/dedis/protobuf"
	"github.com/eliasmpw/Peerster/common"
	"github.com/eliasmpw/Peerster/gossiper"
	"net"
)

const HOP_LIMIT = 10

func main() {
	// Load values passed via flags
	uiPort := flag.String("UIPort", "8080", "Port for the UI client");
	msg := flag.String("msg", "", "Message to be sent");
	dest := flag.String("dest", "", "Destination for the private message")
	file := flag.String("file", "", "File to be indexed by the gossiper")
	request := flag.String("request", "", "Request a chunk or metafile of this hash")
	flag.Parse()
	// Create packet to send
	var packetToSend = gossiper.GossipPacket{}

	if *msg == "" {
		// If there is no message
		if *file != "" && *dest != "" && *request != "" {
			// If it is a download request
			// Convert request hashValue to byte array
			hashValue, err := hex.DecodeString(*request)
			common.CheckError(err)
			packetToSend.DataRequest = &gossiper.DataRequest{
				Origin:      "",
				Destination: *dest,
				HopLimit:    HOP_LIMIT,
				HashValue:   hashValue,
				FileName:    *file,
			}

		} else if *file != "" {
			// If it is a index file request
			// Use Existing SimpleMessage struct with original name file
			simpleWithFilePath := gossiper.SimpleMessage{
				OriginalName:  "file",
				RelayPeerAddr: "file",
				Contents:      *file,
			}
			packetToSend = gossiper.GossipPacket{
				Simple: &simpleWithFilePath,
			}
		}
	} else if *dest == "" {
		// If it is a gossip
		simpleToSend := gossiper.SimpleMessage{
			OriginalName:  "",
			RelayPeerAddr: "",
			Contents:      *msg,
		}
		packetToSend = gossiper.GossipPacket{
			Simple: &simpleToSend,
		}
	} else {
		// If it is a private message (has destination)
		privateToSend := gossiper.PrivateMessage{
			Origin:      "",
			ID:          0,
			Text:        *msg,
			Destination: *dest,
			HopLimit:    HOP_LIMIT,
		}
		packetToSend = gossiper.GossipPacket{
			Private: &privateToSend,
		}
	}

	// Send packet
	content, err := protobuf.Encode(&packetToSend)
	common.CheckError(err)
	addressToSend, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+*uiPort)
	common.CheckError(err)
	udpConnection, err := net.DialUDP("udp4", nil, addressToSend)
	common.CheckError(err)
	udpConnection.Write(content)
	udpConnection.Close()
}
