package gossiper

import (
	"github.com/dedis/protobuf"
	"github.com/eliasmpw/Peerster/common"
	"net"
	"net/http"
	"sync"
	"time"
)

type Gossiper struct {
	address *net.UDPAddr
	conn *net.UDPConn
	Name string
	uiPort string
	uiConn *net.UDPConn
	addressStr string
	isSimple bool
	peersList []string
	allMessages []RumorMessage
	Vc StatusPacket
	channelsListening map[string]chan *PeerStatus
	mutex *sync.Mutex
	routingTable RoutingTable
}

func NewGossiper(uiPort, addressStr, name string, peersList []string, isSimple bool) *Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", addressStr)
	common.CheckError(err)
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	common.CheckError(err)
	uiPort, uiUdpConn := common.StartLocalConnection(uiPort)
	return &Gossiper{
		address: udpAddr,
		conn: udpConn,
		Name: name,
		uiPort: uiPort,
		uiConn: uiUdpConn,
		addressStr: addressStr,
		isSimple: isSimple,
		peersList: peersList,
		allMessages: []RumorMessage{},
		Vc: *NewStatusPacket(name),
		channelsListening:  make(map[string]chan *PeerStatus),
		mutex: &sync.Mutex{},
		routingTable: *NewRoutingTable(),
	}
}

func (gsspr *Gossiper) Serve() {
	// Start goroutines
	var wait sync.WaitGroup
	if gsspr.isSimple {
		wait.Add(2)
		gsspr.StartListeningClientSimple(wait)
		gsspr.StartListeningPeersSimple(wait)
		wait.Wait()
	} else {
		wait.Add(4)
		gsspr.StartListeningClient(wait)
		gsspr.StartListeningGossip(wait)
		gsspr.StartAntiEntropy(wait)
		gsspr.StartServingGUI(wait)
		wait.Wait()
	}
}

func (gsspr *Gossiper) StartListeningClientSimple(wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		defer gsspr.uiConn.Close()
		for {
			buffer := make([]byte, common.BUFFER_SIZE)
			n, sourceAddr, err := gsspr.uiConn.ReadFromUDP(buffer)
			common.CheckError(err)
			onSimpleMessageReceived(gsspr, buffer[:n], sourceAddr, true)
		}
	}()
}

func (gsspr *Gossiper) StartListeningPeersSimple(wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		defer gsspr.conn.Close()
		for {
			buffer := make([]byte, common.BUFFER_SIZE)
			n, sourceAddr, err := gsspr.conn.ReadFromUDP(buffer)
			common.CheckError(err)
			onSimpleMessageReceived(gsspr, buffer[:n], sourceAddr, false)
		}
	}()
}

func (gsspr *Gossiper) StartListeningClient(wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		defer gsspr.uiConn.Close()
		for {
			buffer := make([]byte, common.BUFFER_SIZE)
			n, sourceAddr, err := gsspr.uiConn.ReadFromUDP(buffer)
			common.CheckError(err)
			onMessageReceived(gsspr, buffer[:n], sourceAddr, true)
		}
	}()
}

func (gsspr *Gossiper) StartListeningGossip(wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		defer gsspr.conn.Close()
		for {
			buffer := make([]byte, common.BUFFER_SIZE)
			n, sourceAddr, err := gsspr.conn.ReadFromUDP(buffer)
			common.CheckError(err)
			onMessageReceived(gsspr, buffer[:n], sourceAddr, false)
		}
	}()
}

func (gsspr *Gossiper) StartAntiEntropy(wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		ticker := time.NewTicker(1000 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			randomPeer := GetRandomPeer(gsspr, "")
			if randomPeer != "" {
				logAntiEntropy(randomPeer)
				newPackage := GossipPacket{
					Status: gsspr.Vc.MakeCopy(),
				}
				gsspr.SendPacket(randomPeer, newPackage)
			}
		}
	}()
}

func (gsspr *Gossiper) StartServingGUI(wait sync.WaitGroup) {
	router := createRouteHandlers(gsspr)

	http.Handle("/", router)

	http.ListenAndServe(":" + gsspr.uiPort, router)
}

func (gsspr *Gossiper) SendPacket(destination string, message GossipPacket) {
	// Send gossip packet to destination
	content, err := protobuf.Encode(&message)
	common.CheckError(err)
	addressToSend, err := net.ResolveUDPAddr("udp4", destination)
	common.CheckError(err)
	gsspr.conn.WriteToUDP(content, addressToSend)
}

func (gsspr *Gossiper) addToAllMessagesList(packetReceived RumorMessage) {
	// Store rumor in the list
	messageToSave := RumorMessage{
		Origin: packetReceived.Origin,
		ID:     packetReceived.ID,
		Text:   packetReceived.Text,
	}
	gsspr.allMessages = append(gsspr.allMessages, messageToSave)
}

func (gsspr *Gossiper) FindFromAllMessages(origin string, id uint32) *RumorMessage {
	for _, rumor := range gsspr.allMessages {
		if rumor.Origin == origin && rumor.ID == id {
			return &rumor
		}
	}
	return nil
}
