package gossiper

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/dedis/protobuf"
	"github.com/eliasmpw/Peerster/common"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func onMessageReceived(gsspr *Gossiper, message []byte, sourceAddr *net.UDPAddr, isClient bool) {
	// Decode message
	var packetReceived = GossipPacket{}
	protobuf.Decode(message, &packetReceived)
	if isClient {
		handleClientMessage(gsspr, &packetReceived, sourceAddr);
	} else {
		handleMessage(gsspr, &packetReceived, sourceAddr);
	}
}

func handleClientMessage(gsspr *Gossiper, packetReceived *GossipPacket, sourceAddr *net.UDPAddr) {
	// Handle a message received from the client
	if packetReceived.Simple != nil {
		if packetReceived.Simple.OriginalName == "file" && packetReceived.Simple.RelayPeerAddr == "file" {
			// If it has values of "file" in OriginalName and RelayPeerAddress,
			// handle it as a file index/share request
			fileName := filepath.Base(packetReceived.Simple.Contents)
			absPath, err := filepath.Abs("")
			common.CheckError(err)
			path := absPath +
				string(os.PathSeparator) +
				gsspr.sharedFilesDir +
				fileName
			// Read the file
			fileContent, err := ioutil.ReadFile(path)
			common.CheckError(err)
			fileSize := uint64(len(fileContent))
			// Divide the file into chunks
			chunks := SplitToChunks(fileContent, gsspr.chunkSize)
			// Create hashes for chunks
			hashes := CreateChunkHashes(chunks)
			// Create MetaFile of all hashes
			var metaFile []byte
			for _, hash := range hashes {
				metaFile = append(metaFile, hash...)
			}
			// Create hash of metafile
			h := sha256.New()
			h.Write(metaFile)
			hashValue := h.Sum(nil)
			chunkCount := GetChunkNumber(metaFile)
			completeChunkMap := make([]uint64, chunkCount)
			for i := uint64(0); i < chunkCount; i++ {
				completeChunkMap[i] = uint64(i + 1)
			}
			// Add to the MetaData List
			fmdAux := FileMetaData{
				Origins:   []string{gsspr.Name},
				Name:      fileName,
				Size:      fileSize,
				MetaFile:  metaFile,
				HashValue: hashValue,
				ChunkMap:  completeChunkMap,
			}
			gsspr.metaDataList.Add(fmdAux)

			// Store the file in the disk
			downloadDir := absPath + string(os.PathSeparator) + gsspr.sharedFilesDir
			// Store the file
			WriteFileOnDisk(fileContent, downloadDir, fileName)
			// Store the chunks
			WriteChunksOnDisk(*chunks, gsspr.chunkFilesDir, fileName)
			logFileShared(fileName, hex.EncodeToString(hashValue))
		} else {
			// Else handle as a gossip message
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
	}
	if packetReceived.Private != nil {
		// Handle private message
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
	if packetReceived.DataRequest != nil {
		// Handle download request
		newDataRequest := DataRequest{
			Origin:      gsspr.Name,
			Destination: packetReceived.DataRequest.Destination,
			HopLimit:    packetReceived.DataRequest.HopLimit,
			HashValue:   packetReceived.DataRequest.HashValue,
			FileName:    packetReceived.DataRequest.FileName,
		}
		StartFileDownload(gsspr, newDataRequest)
	}
	if packetReceived.SearchRequest != nil {
		// Handle search request
		newSearchRequest := SearchRequest{
			Origin:   gsspr.Name,
			Budget:   packetReceived.SearchRequest.Budget,
			Keywords: packetReceived.SearchRequest.Keywords,
		}
		StartFileSearch(gsspr, newSearchRequest, true)
	}
}

func handleMessage(gsspr *Gossiper, packetReceived *GossipPacket, sourceAddr *net.UDPAddr) {
	// Handle a message received from a peer
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
	if packetReceived.DataRequest != nil {
		// Handle data request
		ProcessDataRequest(gsspr, *packetReceived.DataRequest, sourceAddr.String())
	}
	if packetReceived.DataReply != nil {
		// Handle data reply
		processDataReply(gsspr, *packetReceived.DataReply, sourceAddr.String())
	}
	if packetReceived.SearchRequest != nil {
		// Handle search request
		ProcessSearchRequest(gsspr, *packetReceived.SearchRequest, sourceAddr.String())
	}
	if packetReceived.SearchReply != nil {
		// Handle data reply
		processSearchReply(gsspr, *packetReceived.SearchReply, sourceAddr.String())
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
