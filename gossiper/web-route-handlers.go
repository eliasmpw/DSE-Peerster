package gossiper

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/eliasmpw/Peerster/common"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"os"
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
	r.HandleFunc("/downloadFile", downloadFileHandler).Methods("POST")
	r.HandleFunc("/searchFile", searchFileHandler).Methods("POST")
	// For streaming
	r.HandleFunc("/streamFileInfo", streamFileInfoHandler).Methods("POST")
	r.PathPrefix("/streaming/").Handler(http.StripPrefix("/streaming/", http.FileServer(http.Dir(myGossiper.sharedFilesDir+"/streaming/"))))
	// For gui file serving
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(dir))))

	return r
}

func streamFileInfoHandler(writer http.ResponseWriter, request *http.Request) {
	var fileSize int64

	reqBody, _ := ioutil.ReadAll(request.Body)
	fileName := string(reqBody[:])
	filePath := myGossiper.sharedFilesDir + "/streaming/" + fileName

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fileSize = 0
	} else {
		file, openError := os.Open(filePath)
		fileInfo, statError := file.Stat()
		if openError != nil || statError != nil {
			fmt.Println("Error while trying to open file", filePath, openError, statError)
			return
		}

		fileSize = fileInfo.Size()
	}

	response := File{
		Name: fileName,
		Size: fileSize,
	}

	json.NewEncoder(writer).Encode(response)
	request.Body.Close()
}

func newMessageHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	var packetReceived GossipPacket
	json.Unmarshal(rawContent, &packetReceived)
	request.Body.Close()
	if &packetReceived == nil {
		return
	}

	handleClientMessage(myGossiper, &packetReceived, myGossiper.address)
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

	handleClientMessage(myGossiper, &packetReceived, myGossiper.address)
}

func shareFileHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	request.Body.Close()

	handleClientMessage(myGossiper, &GossipPacket{
		Simple: &SimpleMessage{
			OriginalName:  "file",
			RelayPeerAddr: "file",
			Contents:      string(rawContent[:]),
		},
	}, myGossiper.address)
}

func downloadFileHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	var packetReceived GossipPacket
	json.Unmarshal(rawContent, &packetReceived)
	handleClientMessage(myGossiper, &GossipPacket{
		DataRequest: packetReceived.DataRequest,
	}, myGossiper.address)
	request.Body.Close()
}

func searchFileHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	var packetReceived GossipPacket
	json.Unmarshal(rawContent, &packetReceived)
	response, err := json.Marshal(StartFileSearch(myGossiper, SearchRequest{
		Origin:   myGossiper.Name,
		Budget:   packetReceived.SearchRequest.Budget,
		Keywords: packetReceived.SearchRequest.Keywords,
	}, false))
	common.CheckError(err)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
	request.Body.Close()
}
