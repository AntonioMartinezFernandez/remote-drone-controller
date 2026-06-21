package main

import (
	"fmt"
	"net"
)

const udpPort = ":4210"

func main() {
	addr, err := net.ResolveUDPAddr("udp", udpPort)
	if err != nil {
		fmt.Println("resolve error:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("listen error:", err)
		return
	}
	defer conn.Close()

	fmt.Println("UDP server listening on", udpPort)

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("read error:", err)
			continue
		}
		fmt.Printf("Received from %s: %s\n", remoteAddr, string(buf[:n]))
	}
}
