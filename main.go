package main

import (
	"flag"
	"github.com/eliasmpw/Peerster/gossiper"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const SHARED_FILES_DIR = "./_SharedFiles/"
const CHUNK_FILES_DIR = "./_SharedFiles/Chunks/"
const DOWNLOADED_FILES_DIR = "./_Downloads/"
const HOP_LIMIT = 10
const HASH_SIZE = 256
const CHUNK_SIZE = 8192
const MAX_SEARCH_BUDGET = 32
const SEARCH_MATCHES_THRESHOLD = 2

var myGossiper *gossiper.Gossiper;

func main() {
	// Initialize random number generator
	randomGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Load values passed via flags
	uiPort := flag.String("UIPort", "8080", "Port for the UI client");
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "IP:port for the gossiper")
	name := flag.String("name", "Peer"+strconv.Itoa(50000+randomGenerator.Intn(9999)),
		"Name of the gossiper")
	peers := flag.String("peers", "", "Comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "Run gossiper in simple broadcast mode")
	rTimer := flag.Int("rtimer", 0, "Route rumors sending period in seconds, 0 to disable sending of route rumors (default 0)")
	flag.Parse()
	var peersSlice []string
	if *peers == "" {
		peersSlice = []string{}
	} else {
		peersSlice = strings.Split(*peers, ",")
	}

	// Start gossiper
	myGossiper = gossiper.NewGossiper(
		*uiPort,
		*gossipAddr,
		*name,
		peersSlice,
		*simple,
		*rTimer,
		SHARED_FILES_DIR,
		CHUNK_FILES_DIR,
		DOWNLOADED_FILES_DIR,
		HOP_LIMIT,
		HASH_SIZE,
		CHUNK_SIZE,
		MAX_SEARCH_BUDGET,
		SEARCH_MATCHES_THRESHOLD,
	)
	myGossiper.Serve()
}
