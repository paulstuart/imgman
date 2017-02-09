package main

import (
	"fmt"
	"net"
	"strings"
)

func udpReader(c chan []byte, closer chan struct{}, fn func(string)) {
	for {
		select {
		case buff := <-c:
            //fmt.Println("UREAD:", string(buff))

			lines := strings.Split(string(buff), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if len(line) > 0 {
					fn(line) //fmt.Println("LINE:", line)
				}
			}
		case _ = <-closer:
			break
		}
	}
}

func udpServer(port int) {
	bind := fmt.Sprintf(":%d", port)
	ServerAddr, err := net.ResolveUDPAddr("udp", bind)
	if err != nil {
		panic(err)
	}

	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		panic(err)
	}
	defer ServerConn.Close()

	for {
	    buf := make([]byte, 4096)
		n, _, err := ServerConn.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("Error: ", err)
		}
		udpChan <- buf[:n]
	}
}
