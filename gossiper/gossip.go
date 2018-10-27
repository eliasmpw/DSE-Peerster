package gossiper

import (
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
		if packetReceived.Simple == nil {
			return
		}
		newPackage := GossipPacket{
			Rumor: &RumorMessage{
				Origin: gsspr.Name,
				ID:     gsspr.Vc.GetNextId(gsspr.Name),
				Text:   packetReceived.Simple.Contents,
			},
		}
		ok := gsspr.Vc.Update(newPackage.Rumor.Origin, newPackage.Rumor.ID)
		if ok {
			gsspr.addToAllMessagesList(*newPackage.Rumor)
			logClientMessage(*packetReceived)
			randomPeer := GetRandomPeer(gsspr, "")
			if randomPeer != "" {
				RumorMonger(gsspr, randomPeer, newPackage)
			}
		}
	} else {
		if packetReceived.Rumor == nil && packetReceived.Status == nil {
			return
		}
		if packetReceived.Rumor != nil {
			addPeerToList(gsspr, sourceAddr.String())
			ok := gsspr.Vc.Update(packetReceived.Rumor.Origin, packetReceived.Rumor.ID)
			if ok {
				gsspr.addToAllMessagesList(*packetReceived.Rumor)
				logRumorMessage(*packetReceived, sourceAddr.String())
				logPeers(gsspr)
				newPackage := GossipPacket{
					Status: gsspr.Vc.MakeCopy(),
				}
				gsspr.SendPacket(sourceAddr.String(), newPackage)
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
	gsspr.SendPacket(destPeer, packet)
	logMongering(destPeer)
	channelId := generateChannelListenId(destPeer, packet.Rumor.Origin, packet.Rumor.ID + 1)
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
			case <- channelListen:
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
			case <- timer.C:
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

