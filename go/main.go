package main

import (
	"log"
	"net"
	"os"

	"github.com/pion/turn/v2"
)

func valueOrDefault(val, default_ string) string {
	if val == "" {
		return default_
	}
	return val
}

var hostname = valueOrDefault(os.Getenv("V_HOSTNAME"), "localhost")

// Replace it with IP address of network interface.
var relayIP = valueOrDefault(os.Getenv("RELAY_ADDR"), "192.168.0.39")

var listenAddr = valueOrDefault(os.Getenv("RELAY_ADDR"), "0.0.0.0:3478")

func StartStunTurn(realm, relayIP, listenAddr string) {

	udpListener, err := net.ListenPacket("udp4", "0.0.0.0:3478")
	if err != nil {
		log.Panicf("Failed to create TURN server listener: %s", err)
	}

	s, err := turn.NewServer(turn.ServerConfig{
		Realm: realm,
		// Set AuthHandler callback
		// This is called everytime a user tries to authenticate with the TURN server
		// Return the key for that user, or false when no user is found
		AuthHandler: func(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
			// Authenticating everyone
			return turn.GenerateAuthKey(username, realm, "thiskey"), true
		},
		// PacketConnConfigs is a list of UDP Listeners and the configuration around them
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(relayIP), // Claim that we are listening on IP passed by user (This should be your Public IP)
					Address:      "0.0.0.0",            // But actually be listening on every interface
				},
			},
		},
	})

	if err != nil {
		log.Panicf("Failed to create TURN server listener: %s", err)
	}

	log.Printf("Serving addr (%s') :: IP relay('%s)", listenAddr, relayIP)

	sigs := make(chan struct{}, 1)
	<-sigs

	if err = s.Close(); err != nil {
		log.Panic(err)
	}
}

func main() {
	StartStunTurn(hostname, relayIP, listenAddr)
}
