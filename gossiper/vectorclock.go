package gossiper

import (
	"sync"
)

type StatusPacket struct {
	Want []PeerStatus
	mutex *sync.Mutex
}

func NewStatusPacket(name string) *StatusPacket {
	var want []PeerStatus
	// Register own clock
	want = append(want, PeerStatus{
		Identifier: name,
		NextID: 1,
	})
	return &StatusPacket{
		Want: want,
		mutex: &sync.Mutex{},
	}
}

func (sp *StatusPacket) Length() int {
	defer sp.mutex.Unlock()
	sp.mutex.Lock()
	length := len(sp.Want)
	return length
}

func (sp *StatusPacket) GetIndex(i int) PeerStatus{
	defer sp.mutex.Unlock()
	sp.mutex.Lock()
	status := sp.Want[i]
	return status
}

func (sp *StatusPacket) GetNextId(name string) uint32 {
	defer sp.mutex.Unlock()
	sp.mutex.Lock()
	for _, status := range sp.Want {
		if status.Identifier == name {
			NextID := status.NextID
			return NextID
		}
	}
	// If clock doesn't exist, create a new one
	sp.Want = append(sp.Want, PeerStatus{
		Identifier: name,
		NextID:     1,
	})
	return 1
}

func (sp *StatusPacket) Add(name string) {
	if sp.GetNextId(name) <= 0 {
		defer sp.mutex.Unlock()
		sp.mutex.Lock()
		sp.Want = append(sp.Want, PeerStatus{
			Identifier: name,
			NextID:     1,
		})
	}
}

func (sp *StatusPacket) Update(name string, id uint32) bool{
	if id != sp.GetNextId(name) {
		return false
	}
	sp.mutex.Lock()
	for index, status := range sp.Want {
		if status.Identifier == name {
			sp.Want[index].NextID++;
			sp.mutex.Unlock()
			return true
		}
	}
	// If clock doesn't exist, create a new one
	sp.Want = append(sp.Want, PeerStatus{
		Identifier: name,
		NextID:     2,
	})
	sp.mutex.Unlock()
	return true
}

func (sp *StatusPacket) MakeCopy() *StatusPacket {
	sp.mutex.Lock()
	Copy := make([]PeerStatus, len(sp.Want))
	copy(Copy, sp.Want)
	sp.mutex.Unlock()
	copyStatusPacket := StatusPacket{
		Want: Copy,
		mutex: &sync.Mutex{},
	}
	return &copyStatusPacket
}

func CompareVectorClocks(gsspr *Gossiper, sourceAddr string, status StatusPacket) {
	isBehind := false
	isSameState := true
	wanting := status.Want
	status.mutex = &sync.Mutex{}

	// Make a copy of vector clock for evaluation
	copy := (&gsspr.Vc).MakeCopy()

	for _, theirStatus := range wanting {
		myId := gsspr.Vc.GetNextId(theirStatus.Identifier)

		if myId < theirStatus.NextID {
			isBehind = true
			isSameState = false
			break
		}
	}

	if isBehind {
		gsspr.sendGossipQueue <- &QueuedMessage{
			packet: GossipPacket{
				Status: copy,
			},
			destination: sourceAddr,
		}
	}

	for i := 0; i < gsspr.Vc.Length(); i++ {
		auxStatus := gsspr.Vc.GetIndex(i)
		myState := auxStatus.NextID
		theirState := status.GetNextId(auxStatus.Identifier)

		if myState > theirState {
			rumorToSend := gsspr.FindFromAllRumorMessages(auxStatus.Identifier, theirState)
			if rumorToSend != nil {
				newPackage := GossipPacket{
					Rumor: rumorToSend,
				}
				go RumorMonger(gsspr, sourceAddr, newPackage)
			}
		}
	}

	if isSameState {
		LogSync(sourceAddr)
	}
	return
}
