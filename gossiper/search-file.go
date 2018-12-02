package gossiper

import (
	"bytes"
	"crypto/sha256"
	"github.com/eliasmpw/Peerster/common"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Structure containing slice of search data and a mutex
type SearchList struct {
	searchData []SearchData
	mutex      *sync.Mutex
}

func NewSearchList() *SearchList {
	return &SearchList{
		searchData: make([]SearchData, 0),
		mutex:      &sync.Mutex{},
	}
}

// Add Search Data to List
func (sl *SearchList) Add(newSearchData SearchData) {
	sl.mutex.Lock()
	sl.searchData = append(sl.searchData, newSearchData)
	sl.mutex.Unlock()
}

// Get the FileMetaData by file hash
func (sl *SearchList) GetByHash(hash []byte) *FileMetaData {
	sl.mutex.Lock()
	for _, sd := range sl.searchData {
		for _, fmd := range sd.MetaDataList {
			if bytes.Equal(fmd.HashValue, hash) {
				sl.mutex.Unlock()
				return &fmd
			}
		}
	}
	sl.mutex.Unlock()
	return nil
}

type RecentSearches struct {
	SearchKeywords []string
	mutex          *sync.Mutex
}

func NewRecentSearches() *RecentSearches {
	return &RecentSearches{
		SearchKeywords: make([]string, 0),
		mutex:          &sync.Mutex{},
	}
}

func (rs *RecentSearches) AddSearch(keywords []string) bool {
	strKeywords := strings.Join(keywords[:], ",")
	rs.mutex.Lock();
	for _, auxKeyword := range rs.SearchKeywords {
		if auxKeyword == strKeywords {
			rs.mutex.Unlock();
			return false
		}
	}
	rs.SearchKeywords = append(rs.SearchKeywords, strKeywords)
	rs.mutex.Unlock();
	go func() {
		time.Sleep(500 * time.Millisecond)
		rs.mutex.Lock();
		for i, auxKeyword := range rs.SearchKeywords {
			if auxKeyword == strKeywords {
				rs.SearchKeywords = append(rs.SearchKeywords[:i], rs.SearchKeywords[i+1:]...)
				break;
			}
		}
		rs.mutex.Unlock();
	}()
	return true
}

type SearchData struct {
	Keywords     []string
	MetaDataList []FileMetaData
}

func NewSearchData(keywords []string) SearchData {
	return SearchData{
		Keywords:     keywords,
		MetaDataList: make([]FileMetaData, 0),
	}
}

func StartFileSearch(gsspr *Gossiper, request SearchRequest, autoDownload bool) []FileMetaData {
	// Set initial budget and start searching
	budgetMax := request.Budget
	searchBudget := request.Budget
	validMetaDatas := make([]FileMetaData, 0)
	auxMetaDataList := make([]FileMetaData, 0)
	for len(validMetaDatas) < gsspr.searchMatchesThreshold {
		// First search locally
		searchBudget--
		auxMetaDataList, validMetaDatas = searchFileLocally(gsspr, request.Keywords, true)
		if len(validMetaDatas) >= gsspr.searchMatchesThreshold {
			break
		}
		if searchBudget > 0 {
			numberOfPeers := uint64(len(gsspr.peersList))
			if searchBudget > numberOfPeers {
				// Send messages
				for i, peer := range gsspr.peersList {
					additionalBudget := uint64(0)
					if searchBudget%numberOfPeers <= uint64(i) {
						additionalBudget = 1
					}
					searchReq := SearchRequest{
						Origin:   gsspr.Name,
						Budget:   searchBudget/numberOfPeers + additionalBudget,
						Keywords: request.Keywords,
					}
					sendSingleSearchRequest(gsspr, searchReq, peer)
				}
			} else {
				// Choose a random peer
				generator := rand.New(rand.NewSource(time.Now().UnixNano()))
				initialRandom := uint64(generator.Int63n(int64(len(gsspr.peersList))))
				for i := uint64(0); i < searchBudget; i++ {
					searchReq := SearchRequest{
						Origin:   gsspr.Name,
						Budget:   1,
						Keywords: request.Keywords,
					}
					sendSingleSearchRequest(gsspr, searchReq, gsspr.peersList[(initialRandom+i)%uint64(len(gsspr.peersList))])
				}
			}

			timer := time.NewTimer(time.Millisecond * 1000)
			waitingForSearch := true
			for waitingForSearch {
				select {
				case <-timer.C:
					// If timer runs out
					timer.Stop()
					waitingForSearch = false
					break
				case replySearch := <-gsspr.searchesListening:
					// Received a reply
					// Check if it matches our search keywords
					if len(replySearch.Results) > 0 {
						isAMatch := false
						for _, key := range request.Keywords {
							if strings.Contains(replySearch.Results[0].FileName, key) {
								isAMatch = true;
								break;
							}
						}
						// Process the reply
						if isAMatch {
							auxMetaDataList, validMetaDatas = processSearchContent(gsspr, auxMetaDataList, replySearch, validMetaDatas)
							if len(validMetaDatas) > gsspr.searchMatchesThreshold {
								timer.Stop()
								waitingForSearch = false
							}
						}
					}
				}
			}
		}

		// Try to increase budget if there weren't enough results
		if len(validMetaDatas) < gsspr.searchMatchesThreshold {
			budgetMax = budgetMax * 2
			searchBudget = budgetMax
			if budgetMax > uint64(gsspr.maxSearchBudget) {
				break
			}
		}
	}

	// Request metafiles for all valid search results
	for i, auxMetaData := range validMetaDatas {
		metaFileReq := DataRequest{
			Origin:      gsspr.Name,
			Destination: auxMetaData.Origins[0],
			HopLimit:    uint32(gsspr.hopLimit),
			FileName:    auxMetaData.Name,
			HashValue:   auxMetaData.HashValue,
		}
		// Decrement HopLimit
		metaFileReq.HopLimit -= 1
		if metaFileReq.HopLimit <= 0 {
			continue
		}

		// Get Next Hop and send
		nextHop := gsspr.routingTable.GetAddress(metaFileReq.Destination)
		if nextHop == "" {
			return make([]FileMetaData, 0)
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
		channelValue, exists := gsspr.filesListening[metaFileReplyString]
		gsspr.filesMutex.Unlock()

		if exists && channelValue != nil {
			// It has already been requested by other thread
			return make([]FileMetaData, 0)
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

				if bytes.Equal(replyHash, auxMetaData.HashValue) {
					// We have received the chunk correctly
					received = true

					// Add to metaDataList
					validMetaDatas[i].MetaFile = make([]byte, len(replyMetaFile.Data))
					copy(validMetaDatas[i].MetaFile, replyMetaFile.Data)

				} else {
					// Invalid MetaFile, keep the loop
					continue
				}
			}
		}

		// Close channel
		close(metaFileReplyChannel)
		gsspr.filesMutex.Lock()
		delete(gsspr.filesListening, metaFileReplyString)
		gsspr.filesMutex.Unlock()
	}
	logSearchFinished()
	if len(validMetaDatas) > 0 {
		gsspr.searchList.Add(SearchData{
			Keywords:     request.Keywords,
			MetaDataList: validMetaDatas,
		})
		if autoDownload {
			// Automatic download at the end of the search if the flag was set to true.
			downloadRequest := DataRequest{
				Origin:      gsspr.Name,
				Destination: validMetaDatas[0].Origins[0],
				HopLimit:    uint32(gsspr.hopLimit),
				HashValue:   validMetaDatas[0].HashValue,
				FileName:    validMetaDatas[0].Name,
			}
			StartFileDownload(gsspr, downloadRequest)
		}
	}
	return validMetaDatas
}

func ProcessSearchRequest(gsspr *Gossiper, request SearchRequest, addressReq string) {
	//Check that it is not a duplicate search request
	if gsspr.recentSearches.AddSearch(request.Keywords) {
		// First search locally
		searchBudget := request.Budget
		searchBudget--
		_, validMetaDatas := searchFileLocally(gsspr, request.Keywords, false)
		if len(validMetaDatas) < gsspr.searchMatchesThreshold && searchBudget > 0 {
			numberOfPeers := uint64(len(gsspr.peersList))
			if searchBudget > numberOfPeers {
				// Send messages
				for i, peer := range gsspr.peersList {
					additionalBudget := uint64(0)
					if searchBudget%numberOfPeers <= uint64(i) {
						additionalBudget = 1
					}
					searchReq := SearchRequest{
						Origin:   request.Origin,
						Budget:   searchBudget/numberOfPeers + additionalBudget,
						Keywords: request.Keywords,
					}
					sendSingleSearchRequest(gsspr, searchReq, peer)
				}
			} else {
				// Choose a random peer
				generator := rand.New(rand.NewSource(time.Now().UnixNano()))
				initialRandom := uint64(generator.Int63n(int64(len(gsspr.peersList))))
				for i := uint64(0); i < searchBudget; i++ {
					searchReq := SearchRequest{
						Origin:   request.Origin,
						Budget:   1,
						Keywords: request.Keywords,
					}
					sendSingleSearchRequest(gsspr, searchReq, gsspr.peersList[(initialRandom+i)%uint64(len(gsspr.peersList))])
				}
			}
		}
		if len(validMetaDatas) > 0 {
			nextHop := gsspr.routingTable.GetAddress(request.Origin)
			if nextHop != "" {
				auxSearchResult := []*SearchResult{}
				for _, resultMetaData := range validMetaDatas {
					auxSearchResult = append(auxSearchResult, &SearchResult{
						FileName:     resultMetaData.Name,
						MetafileHash: resultMetaData.HashValue,
						ChunkMap:     resultMetaData.ChunkMap,
						ChunkCount:   GetChunkNumber(resultMetaData.MetaFile),
					})
				}
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet: GossipPacket{
						SearchReply: &SearchReply{
							Origin:      gsspr.Name,
							Destination: request.Origin,
							HopLimit:    uint32(gsspr.hopLimit),
							Results:     auxSearchResult,
						},
					},
					destination: nextHop,
				}
			}
		}
	}
}

func processSearchReply(gsspr *Gossiper, reply SearchReply, addressReq string) {

	if reply.Destination == gsspr.Name {
		// If we are the destination send to channel
		gsspr.searchesListening <- &reply
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
				SearchReply: &reply,
			},
			destination: nextHop,
		}
	}
	return
}

// Look for a match between the keywords and the files in the process
func searchFileLocally(gsspr *Gossiper, keywords []string, logFindings bool) ([]FileMetaData, []FileMetaData) {
	auxMetaDataList := make([]FileMetaData, 0)
	validMetaDataList := make([]FileMetaData, 0)
	gsspr.metaDataList.mutex.Lock()
	for _, metaData := range gsspr.metaDataList.metaDataFiles {
		for _, key := range keywords {
			if strings.Contains(metaData.Name, key) {
				newFinding := expandMetaDataOrigins(metaData)
				auxMetaDataList = append(auxMetaDataList, newFinding)
				validMetaDataList = append(validMetaDataList, newFinding)
				if logFindings {
					logFoundSearchMatch(newFinding.Name, newFinding.Origins[0])
				}
				break;
			}
		}
	}
	gsspr.metaDataList.mutex.Unlock()
	gsspr.fileDownloadsList.mutex.Lock()
	for _, download := range gsspr.fileDownloadsList.fileDownloads {
		for _, key := range keywords {
			if strings.Contains(download.metaData.Name, key) {
				auxMetaDataList = append(auxMetaDataList, expandMetaDataOrigins(download.metaData))
				break;
			}
		}
	}
	gsspr.fileDownloadsList.mutex.Unlock()
	return auxMetaDataList, validMetaDataList
}

func expandMetaDataOrigins(metaData FileMetaData) FileMetaData {
	oldOriginsSize := uint64(len(metaData.Origins))
	newOrigins := []string{}
	chunkNumber := GetChunkNumber(metaData.MetaFile)
	for i := uint64(0); i < chunkNumber; i++ {
		newOrigins = append(newOrigins, metaData.Origins[i%oldOriginsSize])
	}
	return FileMetaData{
		Origins:   newOrigins,
		Name:      metaData.Name,
		Size:      metaData.Size,
		MetaFile:  metaData.MetaFile,
		HashValue: metaData.HashValue,
		ChunkMap:  metaData.ChunkMap,
	}
}

func processSearchContent(gsspr *Gossiper, list []FileMetaData, reply *SearchReply, validMetaDatas []FileMetaData) ([]FileMetaData, []FileMetaData) {
	gsspr.searchesMutex.Lock()
	for _, auxResult := range reply.Results {
		found := false
		for i, metaData := range list {
			if bytes.Equal(auxResult.MetafileHash, metaData.HashValue) {
				found = true
				list[i] = mergeOrigins(reply.Origin, *auxResult, metaData)
				if uint64(len(list[i].ChunkMap)) == auxResult.ChunkCount {
					existing := false
					for _, oldMeta := range validMetaDatas {
						if bytes.Equal(oldMeta.HashValue, list[i].HashValue) {
							existing = true
							break
						}
					}
					if !existing {
						validMetaDatas = append(validMetaDatas, list[i])
						logFoundSearchMatch(metaData.Name, reply.Origin)
					}
				}
				break
			}
		}
		if !found {
			index := len(list)
			auxNewMetaData := resultToNewMetadata(reply.Origin, *auxResult)
			list = append(list, auxNewMetaData)
			if uint64(len(list[index].ChunkMap)) == auxResult.ChunkCount {
				existing := false
				for _, oldMeta := range validMetaDatas {
					if bytes.Equal(oldMeta.HashValue, list[index].HashValue) {
						existing = true
						break
					}
				}
				if !existing {
					validMetaDatas = append(validMetaDatas, list[index])
					logFoundSearchMatch(auxNewMetaData.Name, reply.Origin)
				}
			}
		}
	}
	gsspr.searchesMutex.Unlock()
	return list, validMetaDatas
}

func mergeOrigins(replyOrigin string, result SearchResult, metaData FileMetaData) FileMetaData {
	for _, index := range result.ChunkMap {
		metaData.Origins[index] = replyOrigin
	}
	metaData.ChunkMap = append([]uint64(nil), common.MergeUint64Slices(metaData.ChunkMap, result.ChunkMap)...)

	return metaData
}

func resultToNewMetadata(replyOrigin string, result SearchResult) FileMetaData {
	auxOrigins := []string{}
	for i := uint64(0); i < result.ChunkCount; i++ {
		auxOrigins = append(auxOrigins, "")
	}
	for _, index := range result.ChunkMap {
		auxOrigins[index-1] = replyOrigin
	}
	return FileMetaData{
		Origins:   auxOrigins,
		Name:      result.FileName,
		Size:      0,
		MetaFile:  nil,
		HashValue: result.MetafileHash,
		ChunkMap:  result.ChunkMap,
	}
}

func getChannelName(origin string, keywords []string) string {
	return origin + strings.Join(keywords, ",")
}

func sendSingleSearchRequest(gsspr *Gossiper, searchReq SearchRequest, peer string) {
	// Send to neighbor
	gsspr.sendGossipQueue <- &QueuedMessage{
		packet: GossipPacket{
			SearchRequest: &searchReq,
		},
		destination: peer,
	}
}
