package gossiper

import (
	"sync"
)

type RoutingTable struct {
	table map[string]string
	mutex *sync.Mutex
}

func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		table: make(map[string]string),
		mutex: &sync.Mutex{},
	}
}

func (r *RoutingTable) GetAddress(name string) string {
	r.mutex.Lock()
	address := r.table[name]
	r.mutex.Unlock()
	return address
}

func (r *RoutingTable) RegisterNextHop(name string, address string) {
	if name == "" || address == "" {
		return
	}
	if r.table[name] != address {
		r.mutex.Lock()
		r.table[name] = address
		logRoutingTableUpdate(name, address)
		r.mutex.Unlock()
	}
}
