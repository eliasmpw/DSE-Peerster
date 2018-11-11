package gossiper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/eliasmpw/Peerster/common"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func StartFileDownload(gsspr *Gossiper, request DataRequest) {
	// Check if we already have the MetaData
	metaData := gsspr.metaDataList.GetByHash(request.HashValue)

	if metaData == nil {
		// First request the MetaFile
		metaFileReq := DataRequest{
			Origin:      gsspr.Name,
			Destination: request.Destination,
			HopLimit:    request.HopLimit,
			FileName:    request.FileName,
			HashValue:   request.HashValue,
		}
		// Decrement HopLimit
		metaFileReq.HopLimit -= 1
		if metaFileReq.HopLimit <= 0 {
			return
		}

		// Get Next Hop and send
		nextHop := gsspr.routingTable.GetAddress(metaFileReq.Destination)
		if nextHop == "" {
			return
		}
		if nextHop != "" {
			gsspr.sendGossipQueue <- &QueuedMessage{
				packet: GossipPacket{
					DataRequest: &metaFileReq,
				},
				destination: nextHop,
			}
		}

		// Log that we are downloading the MetaFile
		logDownloadingMetaFile(metaFileReq.FileName, metaFileReq.Destination)

		// and wait for data reply
		metaFileReplyChannel := make(chan *DataReply)

		metaFileReplyString := string(metaFileReq.HashValue)

		gsspr.filesMutex.Lock()
		_, exists := gsspr.filesListening[metaFileReplyString]
		gsspr.filesMutex.Unlock()

		if exists {
			// It has already been requested by other thread
			return
		}

		// Register new channel
		gsspr.filesMutex.Lock()
		gsspr.filesListening[metaFileReplyString] = metaFileReplyChannel
		gsspr.filesMutex.Unlock()

		received := false

		// While not received
		for !received {
			// Set timer
			timer := time.NewTimer(time.Millisecond * 5000)

			select {
			case <-timer.C:
				// If timer runs out
				timer.Stop()
				// Resend
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet: GossipPacket{
						DataRequest: &metaFileReq,
					},
					destination: nextHop,
				}
			case replyMetaFile := <-metaFileReplyChannel:
				// Received a reply
				timer.Stop()

				// Check integrity of reply content
				hash := sha256.New()
				hash.Write(replyMetaFile.Data)
				replyHash := hash.Sum(nil)

				if bytes.Equal(replyHash, request.HashValue) {
					// We have received the chunk correctly
					received = true

					metaData = &FileMetaData{
						Origin:    replyMetaFile.Origin,
						Name:      request.FileName,
						Size:      GetChunkNumber(replyMetaFile.Data),
						MetaFile:  nil,
						HashValue: replyHash,
					}

					// Add to metaDataList
					metaData.MetaFile = make([]byte, len(replyMetaFile.Data))
					copy(metaData.MetaFile, replyMetaFile.Data)
					gsspr.metaDataList.Add(*metaData)

					// Close channel
					close(metaFileReplyChannel)
					gsspr.filesMutex.Lock()
					gsspr.filesListening[metaFileReplyString] = nil
					gsspr.filesMutex.Unlock()

				} else {
					// Invalid MetaFile, keep the loop
					continue
				}
			}
		}
	}
	fmt.Println("inside")
	fmt.Println("inside2")
	fmt.Println("inside3")
	fmt.Println("inside4")
	fmt.Println("inside5")
	fmt.Println("inside6")
	fmt.Println("inside7")
	fmt.Println(gsspr.metaDataList)
	fmt.Println(metaData.HashValue)
	fmt.Println(metaData.Origin)
	fmt.Println(metaData.Name)
	fmt.Println(metaData.MetaFile)
	fmt.Println(metaData.Size)
	fmt.Println(hex.EncodeToString(request.HashValue))

	chunkNumber := GetChunkNumber(metaData.MetaFile)
	fmt.Println("inside222")
	fmt.Println(chunkNumber)

	fileDownload := FileDownload{
		metaData:  *metaData,
		Chunks:    make([][]byte, 0),
		NextChunk: 0,
		LastChunk: chunkNumber,
	}

	// Add new download to list of downloads
	newDownload := gsspr.fileDownloadsList.Add(&fileDownload)
	if !newDownload {
		return
	}

	for index := 0; uint(index) < chunkNumber; index++ {

		// Download chunk in position index
		chunkHash := metaData.GetChunkHash(index)

		// build the request
		chunkReq := DataRequest{
			Origin:      gsspr.Name,
			Destination: request.Destination,
			HopLimit:    request.HopLimit,
			FileName:    request.FileName,
			HashValue:   chunkHash,
		}

		// Decrement Hoplimit
		chunkReq.HopLimit -= 1
		if chunkReq.HopLimit <= 0 {
			return
		}

		// Get nextHop
		nextHop := gsspr.routingTable.GetAddress(chunkReq.Destination)
		if nextHop == "" {
			return
		}

		// Send Packet
		gsspr.sendGossipQueue <- &QueuedMessage{
			packet: GossipPacket{
				DataRequest: &chunkReq,
			},
			destination: nextHop,
		}

		// print same notification
		logDownloadingChunk(chunkReq.FileName, index+1, chunkReq.Destination)

		// Create channels for chunk reply
		chunkReplyChannel := make(chan *DataReply)
		chunkReplyString := string(chunkReq.HashValue)

		gsspr.filesMutex.Lock()
		_, present := gsspr.filesListening[chunkReplyString]
		gsspr.filesMutex.Unlock()

		if present {
			// Another goroutine is already waiting for this data
			return
		}

		// Register
		gsspr.filesMutex.Lock()
		gsspr.filesListening[chunkReplyString] = chunkReplyChannel
		gsspr.filesMutex.Unlock()

		received := false // not yet received

		for !received {

			timer := time.NewTimer(time.Millisecond * 5000)

			select {
			case <-timer.C:
				// If timer runs out
				timer.Stop()
				// Resend packet
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet: GossipPacket{
						DataRequest: &chunkReq,
					},
					destination: nextHop,
				}
			case chunkReply := <-chunkReplyChannel:
				// When we receive the chunk data
				timer.Stop()

				// Check integrity
				hash := sha256.New()
				hash.Write(chunkReply.Data)
				receivedHash := hash.Sum(nil)

				if bytes.Equal(receivedHash, chunkHash) {
					// We received the correct chunk
					received = true

					chunkData := make([]byte, len(chunkReply.Data))
					copy(chunkData, chunkReply.Data)

					fileDownload.NextChunk++
					fileDownload.Chunks = append(fileDownload.Chunks, chunkData)

					close(chunkReplyChannel)

					gsspr.filesMutex.Lock()
					gsspr.filesListening[chunkReplyString] = nil
					gsspr.filesMutex.Unlock()
				} else {
					// Invalid chunk, keep looping
					continue
				}
			}

		}
	}

	// We have all the chunks, reconstruct file
	reconstructedFile := ReconstructFromChunks(&(fileDownload.Chunks))

	// Store the file in the downloads folder
	path, err := filepath.Abs("")
	common.CheckError(err)
	downloadedPath := path + string(os.PathSeparator) + gsspr.downloadedFilesDir
	WriteFileOnDisk(*reconstructedFile, downloadedPath, request.FileName)

	// Log file downloaded succesfully
	logFileReconstructed(request.FileName)

	// store chunks in disk
	WriteChunksOnDisk(fileDownload.Chunks, gsspr.chunkFilesDir, request.FileName)

	// we are done with the download
	gsspr.fileDownloadsList.Remove(&fileDownload)
}

func ProcessDataRequest(gsspr *Gossiper, request DataRequest, addressReq string) {
	if request.Destination == gsspr.Name {
		// If destination equals our name, we are the destination
		// Get nextHop for response
		nextHop := gsspr.routingTable.GetAddress(request.Origin)
		// If no next hop return to sender
		if nextHop == "" {
			nextHop = addressReq
		}

		hash := request.HashValue

		// Search in metaDataList for a entry with the hash
		metaData := gsspr.metaDataList.GetByHash(hash)

		if metaData != nil {
			// If no match, this is a metafile request
			gsspr.sendGossipQueue <- &QueuedMessage{
				packet: GossipPacket{
					DataReply: &DataReply{
						Origin:      gsspr.Name,
						Destination: request.Origin,
						HopLimit:    uint32(gsspr.hopLimit),
						HashValue:   hash,
						Data:        metaData.MetaFile,
					},
				},
				destination: nextHop,
			}
			return
		}

		// Check if we already have the chunk downloaded
		chunkFileName := GetChunkFilename(request.HashValue)
		chunkFilePath := gsspr.chunkFilesDir + chunkFileName
		chunk, err := ioutil.ReadFile(chunkFilePath)

		if err == nil && chunk != nil {
			// If we have the chunk already, send it
			gsspr.sendGossipQueue <- &QueuedMessage{
				packet: GossipPacket{
					DataReply: &DataReply{
						Origin:      gsspr.Name,
						Destination: request.Origin,
						HopLimit:    uint32(gsspr.hopLimit),
						HashValue:   request.HashValue,
						Data:        chunk,
					},
				},
				destination: nextHop,
			}
			return
		}

		// Check in files being downloaded
		chunkProgress := gsspr.fileDownloadsList.GetChunkByHash(hash)
		if chunkProgress != nil {
			// If we have it, send it
			gsspr.sendGossipQueue <- &QueuedMessage{
				packet: GossipPacket{
					DataReply: &DataReply{
						Origin:      gsspr.Name,
						Destination: request.Origin,
						HopLimit:    uint32(gsspr.hopLimit),
						HashValue:   hash,
						Data:        *chunkProgress,
					},
				},
				destination: nextHop,
			}
			return
		}
		return
	}

	// If we are not the destination we just forward to nextHop
	// Decrement hopLimit and drop if less than 0
	request.HopLimit--
	if request.HopLimit <= 0 {
		return
	}

	// Get nextHop from routingTable
	nextHop := gsspr.routingTable.GetAddress(request.Destination)
	if nextHop != "" {
		gsspr.sendGossipQueue <- &QueuedMessage{
			packet: GossipPacket{
				DataRequest: &request,
			},
			destination: nextHop,
		}
	}
	return
}

func processDataReply(gsspr *Gossiper, reply DataReply, addressReq string) {
	if reply.Destination == gsspr.Name {
		// If we are the destination
		dataReplyString := string(reply.HashValue)

		// Create channel
		gsspr.filesMutex.Lock()
		if gsspr.filesListening[dataReplyString] != nil {
			gsspr.filesListening[dataReplyString] <- &reply
		}
		gsspr.filesMutex.Unlock()
		return
	}

	// If we are not the destination we just forward to nextHop
	// Decrement hopLimit and drop if less than 0
	reply.HopLimit--
	if reply.HopLimit <= 0 {
		return
	}

	// Get nextHop from routingTable
	nextHop := gsspr.routingTable.GetAddress(reply.Destination)
	if nextHop != "" {
		gsspr.sendGossipQueue <- &QueuedMessage{
			packet: GossipPacket{
				DataReply: &reply,
			},
			destination: nextHop,
		}
	}
	return
}
