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
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(dir))))

	return r
}

func mainHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "index.html")
}

func bootstrapCssHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "gui/css/bootstrap.min.css")
}

func styleCssHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "gui/css/style.css")
}

func bootstrapJsHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "gui/js/bootstrap.min.js")
}

func jqueryMinHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "gui/js/jquery.min.js")
}

func popperMinHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "gui/js/popper.min.js")
}

func scriptJsHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "gui/js/script.js")
}

func newMessageHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	var packetReceived GossipPacket
	json.Unmarshal(rawContent, &packetReceived)
	if &packetReceived == nil {
		return
	}

	handleMessage(myGossiper, &packetReceived, myGossiper.address, true)
}

func newNodeHandler(writer http.ResponseWriter, request *http.Request) {
	rawContent, _ := ioutil.ReadAll(request.Body)
	request.Body.Close()

	newNode := string(rawContent[:])
	addPeerToList(myGossiper, newNode)
}

func messagesHandler(writer http.ResponseWriter, request *http.Request) {
	response, err := json.Marshal(myGossiper.allMessages)
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