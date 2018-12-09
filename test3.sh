#!/usr/bin/env bash

go build

./Peerster -peers=127.0.0.1:5001 -name=primerito -rtimer=1 -df=1 &
sleep 1
./Peerster -UIPort=8081 -gossipAddr=127.0.0.1:5001 -name=secondOne -peers=127.0.0.1:5000,127.0.0.1:5002 -rtimer=1 -df=2 &
sleep 1
./Peerster -UIPort=8082 -gossipAddr=127.0.0.1:5002 -name=known -rtimer=1 -df=3 &
sleep 1
./Peerster -UIPort=8083 -gossipAddr=127.0.0.1:5003 -name=unknown -peers=127.0.0.1:5002 -rtimer=1 -df=4 &

