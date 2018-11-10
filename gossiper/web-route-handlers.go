package gossiper

import (
	"encoding/json"
	"flag"
	"github.com/eliasmpw/Peerster/common"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

var myGossiper *Gossiper

func createRouteHandlers(gsspr *Gossiper) *mux.Router {
	myGossiper = gsspr
	var dir string

	flag.StringVar(&dir, "./gui/", "./gui/", "The directory to serve files from. Defaults to the current dir")
	flag.Parse()
	r := mux.NewRouter()

	r.HandleFunc("/message", newMessageHandler).Methods("POST")
	r.HandleFunc("/node", newNodeHandler).Methods("POST")
	r.HandleFunc("/message", messagesHandler).Methods("GET")
	r.HandleFunc("/node", nodesHandler).Methods("GET")
	r.HandleFunc("/id", idHandler).Methods("GET")
	r.HandleFunc("/ipAddress", ipAddressHandler).Methods("GET")
	r.HandleFunc("/allNodes", allNodesHandler).Methods("GET")
	r.HandleFunc("/privateMessage", privateMessageHandler).Methods("GET")
	r.HandleFunc("/privateMessage", newPrivateMessageHandler).Methods("POST")
	r.HandleFunc("/shareFile", shareFileHandler).Methods("POST")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(dir))))

	return r
}

func newMessageHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	var packetReceived GossipPacket
	json.Unmarshal(rawContent, &packetReceived)
	request.Body.Close()
	if &packetReceived == nil {
		return
	}

	handleMessageClient(myGossiper, &packetReceived, myGossiper.address)
}

func newNodeHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	request.Body.Close()

	newNode := string(rawContent[:])
	addPeerToList(myGossiper, newNode)
}

func messagesHandler(writer http.ResponseWriter, request *http.Request) {
	filteredMessages := []RumorMessage{}
	for _, message := range myGossiper.allRumorMessages {
		if message.Text != "" {
			filteredMessages = append(filteredMessages, message)
		}
	}
	response, err := json.Marshal(filteredMessages)
	common.CheckError(err)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
}

func nodesHandler(writer http.ResponseWriter, request *http.Request) {
	response, err := json.Marshal(myGossiper.peersList)
	common.CheckError(err)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
}

func idHandler(writer http.ResponseWriter, request *http.Request) {
	response, err := json.Marshal(myGossiper.Name)
	common.CheckError(err)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
}

func ipAddressHandler(writer http.ResponseWriter, request *http.Request) {
	response, err := json.Marshal(myGossiper.addressStr)
	common.CheckError(err)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
}

func allNodesHandler(writer http.ResponseWriter, request *http.Request) {
	response, err := json.Marshal(myGossiper.routingTable.table)
	common.CheckError(err)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
}

func privateMessageHandler(writer http.ResponseWriter, request *http.Request) {
	response, err := json.Marshal(myGossiper.allPrivateMessages)
	common.CheckError(err)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
}

func newPrivateMessageHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	var packetReceived GossipPacket
	json.Unmarshal(rawContent, &packetReceived)
	request.Body.Close()
	if &packetReceived == nil {
		return
	}

	handleMessageClient(myGossiper, &packetReceived, myGossiper.address)
}

func shareFileHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	request.Body.Close()

	handleMessageClient(myGossiper, &GossipPacket{
		Simple: &SimpleMessage{
			OriginalName:  "file",
			RelayPeerAddr: "file",
			Contents:      string(rawContent[:]),
		},
	}, myGossiper.address)
}
