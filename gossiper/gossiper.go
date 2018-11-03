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
	address            *net.UDPAddr
	conn               *net.UDPConn
	Name               string
	uiPort             string
	uiConn             *net.UDPConn
	addressStr         string
	isSimple           bool
	peersList          []string
	allPrivateMessages []PrivateMessage
	allRumorMessages   []RumorMessage
	Vc                 StatusPacket
	channelsListening  map[string]chan *PeerStatus
	mutex              *sync.Mutex
	routingTable       RoutingTable
	routeRumorTimer    int
	sendGossipQueue    chan *QueuedMessage
}

func NewGossiper(uiPort, addressStr, name string, peersList []string, isSimple bool, rTimer int) *Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", addressStr)
	common.CheckError(err)
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	common.CheckError(err)
	uiPort, uiUdpConn := common.StartLocalConnection(uiPort)
	return &Gossiper{
		address:            udpAddr,
		conn:               udpConn,
		Name:               name,
		uiPort:             uiPort,
		uiConn:             uiUdpConn,
		addressStr:         addressStr,
		isSimple:           isSimple,
		peersList:          peersList,
		allPrivateMessages: []PrivateMessage{},
		allRumorMessages:   []RumorMessage{},
		Vc:                 *NewStatusPacket(name),
		channelsListening:  make(map[string]chan *PeerStatus),
		mutex:              &sync.Mutex{},
		routingTable:       *NewRoutingTable(),
		routeRumorTimer:    rTimer,
		sendGossipQueue:    make(chan *QueuedMessage),
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
		wait.Add(6)
		gsspr.StartListeningClient(wait)
		gsspr.StartListeningGossip(wait)
		gsspr.StartGossipSender(wait)
		gsspr.StartRouteRumoring(wait)
		gsspr.StartServingGUI(wait)
		gsspr.StartAntiEntropy(wait)
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
				gsspr.sendGossipQueue <- &QueuedMessage{
					packet:      newPackage,
					destination: randomPeer,
				}
			}
		}
	}()
}

func (gsspr *Gossiper) StartServingGUI(wait sync.WaitGroup) {
	go func() {
		router := createRouteHandlers(gsspr)

		http.Handle("/", router)

		http.ListenAndServe(":"+gsspr.uiPort, router)
	}()
}

func (gsspr *Gossiper) addToAllRumorMessagesList(packetReceived RumorMessage) {
	// Store rumor in the list
	messageToSave := RumorMessage{
		Origin: packetReceived.Origin,
		ID:     packetReceived.ID,
		Text:   packetReceived.Text,
	}
	gsspr.allRumorMessages = append(gsspr.allRumorMessages, messageToSave)
}

func (gsspr *Gossiper) addToAllPrivateMessagesList(packetReceived PrivateMessage) {
	// Store private message in the list
	messageToSave := PrivateMessage{
		Origin:      packetReceived.Origin,
		ID:          packetReceived.ID,
		Text:        packetReceived.Text,
		Destination: packetReceived.Destination,
		HopLimit:    packetReceived.HopLimit,
	}
	gsspr.allPrivateMessages = append(gsspr.allPrivateMessages, messageToSave)
}

func (gsspr *Gossiper) FindFromAllRumorMessages(origin string, id uint32) *RumorMessage {
	for _, rumor := range gsspr.allRumorMessages {
		if rumor.Origin == origin && rumor.ID == id {
			return &rumor
		}
	}
	return nil
}

func (gsspr *Gossiper) StartRouteRumoring(wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		if gsspr.routeRumorTimer != 0 {
			for _, peer := range gsspr.peersList {
				newPackage := GossipPacket{
					Rumor: &RumorMessage{
						Origin: gsspr.Name,
						ID:     gsspr.Vc.GetNextId(gsspr.Name),
						Text:   "",
					},
				}
				if peer != gsspr.addressStr {
					RumorMonger(gsspr, peer, newPackage)
				}
			}
			ticker := time.NewTicker(time.Duration(gsspr.routeRumorTimer) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				randomPeer := GetRandomPeer(gsspr, "")
				if randomPeer != "" {
					newPackage := GossipPacket{
						Rumor: &RumorMessage{
							Origin: gsspr.Name,
							ID:     gsspr.Vc.GetNextId(gsspr.Name),
							Text:   "",
						},
					}
					RumorMonger(gsspr, randomPeer, newPackage)
				}
			}
		}
	}()
}

func (gsspr *Gossiper) StartGossipSender(wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		for qMessage := range gsspr.sendGossipQueue {
			packet := qMessage.packet
			destination := qMessage.destination

			// Send gossip packet to destination
			content, err := protobuf.Encode(&packet)
			common.CheckError(err)
			addressToSend, err := net.ResolveUDPAddr("udp4", destination)
			common.CheckError(err)
			gsspr.conn.WriteToUDP(content, addressToSend)
		}
	}()
}
