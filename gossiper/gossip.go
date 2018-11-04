package gossiper

import (
	"fmt"
	"github.com/dedis/protobuf"
	"github.com/eliasmpw/Peerster/common"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func onMessageReceived(gsspr *Gossiper, message []byte, sourceAddr *net.UDPAddr, isClient bool) {
	// Decode message
	var packetReceived = GossipPacket{}
	protobuf.Decode(message, &packetReceived)
	handleMessage(gsspr, &packetReceived, sourceAddr, isClient);
}

func handleMessage(gsspr *Gossiper, packetReceived *GossipPacket, sourceAddr *net.UDPAddr, isClient bool) {
	if isClient {
		if packetReceived.Simple != nil {
			newPackage := GossipPacket{
				Rumor: &RumorMessage{
					Origin: gsspr.Name,
					ID:     gsspr.Vc.GetNextId(gsspr.Name),
					Text:   packetReceived.Simple.Contents,
				},
			}
			ok := gsspr.Vc.Update(newPackage.Rumor.Origin, newPackage.Rumor.ID)
			if ok {
				gsspr.addToAllRumorMessagesList(*newPackage.Rumor)
				logClientMessage(*packetReceived)
				randomPeer := GetRandomPeer(gsspr, "")
				if randomPeer != "" {
					RumorMonger(gsspr, randomPeer, newPackage)
				}
			}
		}
		if packetReceived.Private != nil {
			fmt.Printf("packettttt %s to deliver to %s\n", packetReceived.Private.Text, packetReceived.Private.Destination)
			newPackage := GossipPacket{
				Private: &PrivateMessage{
					Origin:      gsspr.Name,
					ID:          0,
					Text:        packetReceived.Private.Text,
					Destination: packetReceived.Private.Destination,
					HopLimit:    packetReceived.Private.HopLimit,
				},
			}
			gsspr.addToAllPrivateMessagesList(*newPackage.Private)
			RoutePrivateMessage(gsspr, newPackage)
		}
	} else {
		if packetReceived.Rumor != nil {
			addPeerToList(gsspr, sourceAddr.String())
			ok := gsspr.Vc.Update(packetReceived.Rumor.Origin, packetReceived.Rumor.ID)
			if ok {
				if packetReceived.Rumor.Origin != gsspr.Name && sourceAddr.String() != gsspr.addressStr {
					gsspr.routingTable.RegisterNextHop(packetReceived.Rumor.Origin, sourceAddr.String())
				}
				gsspr.addToAllRumorMessagesList(*packetReceived.Rumor)
				if packetReceived.Rumor.Text != "" {
					logRumorMessage(*packetReceived, sourceAddr.String())
					logPeers(gsspr)
				}
				newPackage := GossipPacket{
					Status: gsspr.Vc.MakeCopy(),
				}
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet:      newPackage,
					destination: sourceAddr.String(),
				}
			}
		}
		if packetReceived.Status != nil {
			addPeerToList(gsspr, sourceAddr.String())
			logStatusMessage(*packetReceived, sourceAddr.String())
			logPeers(gsspr)
			for _, status := range packetReceived.Status.Want {
				// Check if there are status that we were waiting for
				channelListenId := generateChannelListenId(sourceAddr.String(), status.Identifier, status.NextID)
				gsspr.mutex.Lock()
				channel, exists := gsspr.channelsListening[channelListenId]
				if exists && channel != nil {
					select {
					case channel <- &status:
					default:
					}
				}
				gsspr.mutex.Unlock()
			}

			CompareVectorClocks(gsspr, sourceAddr.String(), *packetReceived.Status)
		}
		if packetReceived.Private != nil {
			if packetReceived.Private.Destination == gsspr.Name {
				logPrivateMessage(*packetReceived)
				gsspr.addToAllPrivateMessagesList(*packetReceived.Private)
			} else {
				RoutePrivateMessage(gsspr, *packetReceived)
			}
		}
	}
}

func GetRandomPeer(gsspr *Gossiper, ignore string) string {
	// Choose a random peer
	if len(gsspr.peersList) == 0 || len(gsspr.peersList) == 1 && gsspr.peersList[0] == ignore {
		return ""
	}
	generator := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomPeer := ""
	for randomPeer == "" || randomPeer == ignore {
		randomPeer = gsspr.peersList[generator.Intn(len(gsspr.peersList))]
	}
	return randomPeer
}

func RumorMonger(gsspr *Gossiper, destPeer string, packet GossipPacket) {
	// Start mongering with a peer
	gsspr.sendGossipQueue <- &QueuedMessage{
		packet:      packet,
		destination: destPeer,
	}
	logMongering(destPeer)
	channelId := generateChannelListenId(destPeer, packet.Rumor.Origin, packet.Rumor.ID+1)
	channelListen := make(chan *PeerStatus)
	gsspr.mutex.Lock()
	_, ok := gsspr.channelsListening[channelId]
	gsspr.mutex.Unlock()
	if ok {
		// Already waiting, close channel
		close(channelListen)
		return
	}
	gsspr.mutex.Lock()
	gsspr.channelsListening[channelId] = channelListen
	gsspr.mutex.Unlock()

	go func() {
		timer := time.NewTimer(1000 * time.Millisecond)
		select {
		case <-channelListen:
			timer.Stop()
			gsspr.mutex.Lock()
			close(channelListen)
			gsspr.channelsListening[channelId] = nil
			delete(gsspr.channelsListening, channelId)
			gsspr.mutex.Unlock()
			if common.FlipCoin() {
				randomPeer := GetRandomPeer(gsspr, destPeer)
				if randomPeer != "" {
					LogFlippedCoin(randomPeer)
					go RumorMonger(gsspr, randomPeer, packet)
				}
			}
			return
		case <-timer.C:
			timer.Stop()
			gsspr.mutex.Lock()
			close(channelListen)
			gsspr.channelsListening[channelId] = nil
			delete(gsspr.channelsListening, channelId)
			gsspr.mutex.Unlock()
			if common.FlipCoin() {
				randomPeer := GetRandomPeer(gsspr, destPeer)
				if randomPeer != "" {
					LogFlippedCoin(randomPeer)
					go RumorMonger(gsspr, randomPeer, packet)
				}
			}
			return
		}
	}()
	return
}

func generateChannelListenId(destPeer string, origin string, id uint32) string {
	return origin + "," + destPeer + "," + strconv.Itoa(int(id))
}

func addPeerToList(gsspr *Gossiper, addr string) bool {
	// Add a peer to the list of peers
	found := false
	for _, peer := range gsspr.peersList {
		if peer == addr {
			found = true
		}
	}
	if !found {
		gsspr.peersList = append(gsspr.peersList, addr)
		return true
	}
	return false
}

func RoutePrivateMessage(gsspr *Gossiper, packet GossipPacket) {
	packet.Private.HopLimit--
	nextHop := gsspr.routingTable.GetAddress(packet.Private.Destination)

	if packet.Private.HopLimit > 0 && nextHop != "" {
		gsspr.sendGossipQueue <- &QueuedMessage{
			packet:      packet,
			destination: nextHop,
		}
	}
}
