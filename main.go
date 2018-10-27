package main

import (
	"flag"
	"github.com/eliasmpw/Peerster/gossiper"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var myGossiper *gossiper.Gossiper;

func main() {
	// Initialize random number generator
	randomGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Load values passed via flags
	uiPort := flag.String("UIPort", "8080", "Port for the UI client");
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "IP:port for the gossiper")
	name := flag.String("name", "Peer" + strconv.Itoa(50000 + randomGenerator.Intn(9999)),
		"Name of the gossiper")
	peers := flag.String("peers", "", "Comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "Run gossiper in simple broadcast mode")
	flag.Parse()
	var peersSlice []string
	if *peers == "" {
		peersSlice = []string{}
	} else {
		peersSlice = strings.Split(*peers, ",")
	}

	// Start gossiper
	myGossiper = gossiper.NewGossiper(*uiPort, *gossipAddr, *name, peersSlice, *simple)
    myGossiper.Serve()
}
