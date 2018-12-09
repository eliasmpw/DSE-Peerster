package gossiper

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"time"
)

func startMining(gsspr *Gossiper) {
	for {
		// Start recording time for mining to give an equal delay later
		start := time.Now()
		// Generate random Nonce
		gsspr.miningBlock.Nonce = randomNonce()
		auxBlock := gsspr.miningBlock
		// Check proof of work, if not valid keep mining
		for !isBlockPOWValid(auxBlock) {
			gsspr.miningBlock.Nonce = randomNonce()
			auxBlock = gsspr.miningBlock
		}
		// Log that we mined a new block
		logFoundBlock(auxBlock)
		elapsed := time.Since(start)
		// Insert to BlockChain
		inserted := InsertBlockToBlockChain(gsspr, auxBlock)
		if inserted {
			// If it is first block sleep 5 seconds, else sleep for twice the mining time (following the HW3 specifications)
			if len(gsspr.blockChain.forks) == 1 {
				PublishBlockWithDelays(gsspr, time.Duration(5*time.Second), auxBlock)
			} else {
				PublishBlockWithDelays(gsspr, time.Duration(elapsed.Seconds()*2)*time.Second, auxBlock)
			}
		}
	}
}

func randomNonce() [32]byte {
	randomGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
	var nonce [32]byte
	for i, _ := range nonce {
		nonce[i] = byte(randomGenerator.Intn(255))
	}
	return nonce
}

func isBlockPOWValid(miningBlock Block) bool {
	blockHash := miningBlock.Hash()
	// check that the first 16 bits equal zero
	return blockHash[0] == 0 && blockHash[1] == 0
}

func broadcastNewFile(gsspr *Gossiper, file File) {
	for _, peer := range gsspr.peersList {
		if peer != gsspr.addressStr {
			gsspr.sendGossipQueue <- &QueuedMessage{
				packet: GossipPacket{
					TxPublish: &TxPublish{
						File:     file,
						HopLimit: uint32(gsspr.hopLimit), // This will be 10
					},
				},
				destination: peer,
			}
		}
	}
}

func processTransactionReceived(gsspr *Gossiper, transaction TxPublish, sourceAddress string) {
	inserted := InsertTransactionToMiningBlock(gsspr, transaction)
	if inserted {
		transaction.HopLimit--
		if transaction.HopLimit > 0 {
			for _, peer := range gsspr.peersList {
				if peer != gsspr.addressStr && peer != sourceAddress {
					gsspr.sendGossipQueue <- &QueuedMessage{
						packet: GossipPacket{
							TxPublish: &transaction,
						},
						destination: peer,
					}
				}
			}
		}
	}
}

func InsertTransactionToMiningBlock(gsspr *Gossiper, transaction TxPublish) bool {
	gsspr.blockMutex.Lock()
	// Check that the transaction is not already part of our current list of transactions
	for _, auxTransaction := range gsspr.miningBlock.Transactions {
		if auxTransaction.File.Name == transaction.File.Name {
			gsspr.blockMutex.Unlock()
			return false
		}
	}
	// Check that the transaction is not part of our current longest chain
	if FileNameExistsInRoute(&gsspr.blockChain, transaction.File.Name, gsspr.currentForkRoute) {
		return false
	}
	// Insert the transaction to the block we are currently mining
	gsspr.miningBlock.Transactions = append(gsspr.miningBlock.Transactions, transaction)
	gsspr.blockMutex.Unlock()
	return true
}

func processBlockPublishReceived(gsspr *Gossiper, blockPublish BlockPublish, sourceAddress string) {
	inserted := InsertBlockToBlockChain(gsspr, blockPublish.Block)
	if inserted {
		blockPublish.HopLimit--
		if blockPublish.HopLimit > 0 {
			for _, peer := range gsspr.peersList {
				if peer != gsspr.addressStr && peer != sourceAddress {
					gsspr.sendGossipQueue <- &QueuedMessage{
						packet: GossipPacket{
							BlockPublish: &blockPublish,
						},
						destination: peer,
					}
				}
			}
		}
	}
}

func InsertBlockToBlockChain(gsspr *Gossiper, block Block) bool {
	// Check that the hash is valid
	valid := isBlockPOWValid(block)
	hash := block.Hash()
	hashString := hex.EncodeToString(hash[:])
	if !valid {
		return false
	}
	gsspr.blockMutex.Lock()
	// If this is the first chain we insert it
	if len(gsspr.blockChain.forks) == 0 {
		gsspr.blockChain.forks[hashString] = &BlockChainNode{
			block: block,
			forks: make(map[string]*BlockChainNode),
		}
	} else {
		// Check that prevhash exists in the blockchain (except if it is the first block)
		prevHashString := hex.EncodeToString(block.PrevHash[:])
		exists, route := FindRouteToBlockByHash(&gsspr.blockChain, prevHashString, []string{})
		if exists {
			inserted := InsertToChainByRoute(&gsspr.blockChain, Block{
				PrevHash:     block.PrevHash,
				Nonce:        block.Nonce,
				Transactions: block.Transactions,
			}, route)
			if !inserted {
				gsspr.blockMutex.Unlock()
				return false
			}
		} else {
			gsspr.blockMutex.Unlock()
			return false
		}
	}
	// Search for new longest fork
	maxSize, maxRoute := GetLongestFork(&gsspr.blockChain, 0, []string{})
	// if there is a new longest fork
	if maxSize > len(gsspr.currentForkRoute) {
		//Get size of rewind
		rewindSize := len(GetRewindedRoute(gsspr.currentForkRoute, maxRoute))
		// if rewind is 0 then we are continuing the current fork else log a new longer fork
		if rewindSize != 0 {
			logForkLonger(rewindSize)
		}
		gsspr.currentForkRoute = append([]string{}, maxRoute...)
		auxNewBlockList := GetBlocksFromRoute(&gsspr.blockChain, maxRoute, []Block{})
		gsspr.currentFork = append([]Block{}, auxNewBlockList[1:]...)
	} else {
		logForkShorter(block)
	}
	// Set prev hash of ongoing mining block to this new one
	for i, auxByte := range gsspr.currentFork[len(gsspr.currentFork)-1].Hash() {
		gsspr.miningBlock.PrevHash[i] = auxByte
	}
	// remove ongoing transactions that are already part of the new chain
	newTransactions := gsspr.miningBlock.Transactions[:0]
	for _, ownTransaction := range gsspr.miningBlock.Transactions {
		found := FileNameExistsInRoute(&gsspr.blockChain, ownTransaction.File.Name, gsspr.currentForkRoute)
		if !found {
			newTransactions = append(newTransactions, ownTransaction)
		}
	}
	gsspr.miningBlock.Transactions = newTransactions
	logBlockIntegratedChain(gsspr.currentFork)
	gsspr.blockMutex.Unlock()
	return true
}

// Hash Block
func (b *Block) Hash() (out [32]byte) {
	h := sha256.New()
	h.Write(b.PrevHash[:])
	h.Write(b.Nonce[:])
	binary.Write(h, binary.LittleEndian,
		uint32(len(b.Transactions)))
	for _, t := range b.Transactions {
		th := t.Hash()
		h.Write(th[:])
	}
	copy(out[:], h.Sum(nil))
	return
}

// Hash TxPublish
func (t *TxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h, binary.LittleEndian,
		uint32(len(t.File.Name)))
	h.Write([]byte(t.File.Name))
	h.Write(t.File.MetafileHash)
	copy(out[:], h.Sum(nil))
	return
}

type BlockChainNode struct {
	block Block
	forks map[string]*BlockChainNode
}

func NewBlockChain() BlockChainNode {
	return BlockChainNode{
		block: Block{},
		forks: make(map[string]*BlockChainNode),
	}
}

func FindRouteToBlockByHash(node *BlockChainNode, hash string, route []string) (bool, []string) {
	auxHash := node.block.Hash()
	if hex.EncodeToString(auxHash[:]) == hash {
		return true, route
	}
	route = append(route, "")
	for key, auxNode := range node.forks {
		route[len(route)-1] = key
		exists, auxRoute := FindRouteToBlockByHash(auxNode, hash, route)
		if exists {
			return true, auxRoute
		}
	}
	route = route[:len(route)-1]
	return false, route
}

func InsertToChainByRoute(node *BlockChainNode, newBlock Block, route []string) bool {
	if len(route) == 0 {
		auxHash := newBlock.Hash()
		node.forks[hex.EncodeToString(auxHash[:])] = &BlockChainNode{
			block: newBlock,
			forks: make(map[string]*BlockChainNode),
		}
		return true
	}
	_, exists := node.forks[route[0]]
	if exists {
		return InsertToChainByRoute(node.forks[route[0]], newBlock, route[1:])
	} else {
		return false
	}
}

func GetLongestFork(node *BlockChainNode, size int, route []string) (int, []string) {
	if len(node.forks) == 0 {
		return size + 1, route
	}
	maxSize := 0
	maxRoute := route
	auxRoute := append(route, "")
	for key, _ := range node.forks {
		auxRoute[len(auxRoute)-1] = key
		auxSize, auxRoute := GetLongestFork(node.forks[key], size+1, auxRoute)
		if auxSize > maxSize {
			maxSize = auxSize
			maxRoute = append([]string{}, auxRoute...)
		}
	}
	return maxSize, maxRoute
}

func GetBlocksFromRoute(node *BlockChainNode, route []string, blockList []Block) []Block {
	if len(route) == 0 {
		return append(blockList, node.block)
	}
	_, exists := node.forks[route[0]]
	auxList := append(blockList, node.block)
	if exists {
		return GetBlocksFromRoute(node.forks[route[0]], route[1:], auxList)
	} else {
		return auxList
	}
}

func FileNameExistsInRoute(node *BlockChainNode, fileName string, route []string) bool {
	for _, auxTrans := range node.block.Transactions {
		if auxTrans.File.Name == fileName {
			return true
		}
	}
	if len(route) == 0 {
		return false
	} else {
		_, exists := node.forks[route[0]]
		if exists {
			return FileNameExistsInRoute(node.forks[route[0]], fileName, route[1:])
		} else {
			return false
		}
	}
}

func GetRewindedRoute(oldRoute []string, newRoute []string) []string {
	for i, auxHash := range oldRoute {
		if auxHash != newRoute[i] {
			return oldRoute[i:]
		}
	}
	return []string{}
}

func PublishBlockWithDelays(gsspr *Gossiper, delay time.Duration, block Block) {
	go func() {
		time.Sleep(delay)
		for _, peer := range gsspr.peersList {
			if peer != gsspr.addressStr {
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet: GossipPacket{
						BlockPublish: &BlockPublish{
							Block:    block,
							HopLimit: uint32(gsspr.hopLimit * 2), //This will be 20 as specified in HW3.pdf
						},
					},
					destination: peer,
				}
			}
		}
	}()
}
