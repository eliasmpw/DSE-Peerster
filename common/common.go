package common

import (
	"math/rand"
	"net"
	"strconv"
)

const BUFFER_SIZE = 65535

func StartLocalConnection(port string) (portString string, udpConn *net.UDPConn) {
	var udpAddr *net.UDPAddr
	var err error
	successful := false
	for !successful {
		udpAddr, err = net.ResolveUDPAddr("udp4", "127.0.0.1:" + port)
		CheckError(err)
		udpConn, err = net.ListenUDP("udp4", udpAddr)
		if err == nil {
			successful = true
		} else {
			intAux, err := strconv.Atoi(port)
			CheckError(err)
			port = strconv.Itoa(intAux + 1)
		}
	}
	return port, udpConn
}

func CheckError(e error) {
	if e != nil {
		//log.Fatal(e)
		panic(e)
	}
}

func FlipCoin() bool {
	return rand.Int() % 2 == 1
}
