package common

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

const BUFFER_SIZE = 65535

// Create a UDP connection
func StartLocalConnection(port string) (portString string, udpConn *net.UDPConn) {
	var udpAddr *net.UDPAddr
	var err error
	successful := false
	for !successful {
		udpAddr, err = net.ResolveUDPAddr("udp4", "127.0.0.1:"+port)
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

// Check Error
func CheckError(e error) {
	if e != nil {
		//log.Fatal(e)
		panic(e)
	}
}

func FlipCoin() bool {
	return rand.Int()%2 == 1
}

func MergeUint64Slices(left, right []uint64) []uint64 {
	into := make([]uint64, 0, len(left)+len(right))
	if len(left) == 0 {
		return append(into, right...)
	}
	if len(right) == 0 {
		return append(into, left...)
	}

	rightLast := 0
	for _, lv := range left {
		for _, rv := range right[rightLast:] {
			if lv == rv {
				rightLast += 1
				continue
			}
			if lv < rv {
				break
			}
			into = append(into, rv)
			rightLast += 1
		}
		into = append(into, lv)
	}
	into = append(into, right[rightLast:]...)
	return into
}

func IntTo32Hex(value int) [32]byte {
	var auxArray [32]byte
	auxSlice, _ := hex.DecodeString(fmt.Sprintf("%064x", value))
	copy(auxArray[:], auxSlice)

	return auxArray
}

func UInt64ArrayToString(myArray []uint64, separator string) string {
	return strings.Trim(strings.Replace(fmt.Sprint(myArray), " ", separator, -1), "[]")
}
