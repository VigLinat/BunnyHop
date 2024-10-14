package main

import (
    "flag"
    "fmt"
    "log"
    "net"
    "os"
    "time"
)

var port = flag.String("p","50160", "http server port")
var addr = flag.String("a", "localhost", "http server ip address")

func main() {
	flag.Parse()
    hostAddr := fmt.Sprintf("%s:%s", *addr, *port)
	listener, err := net.Listen("tcp4", hostAddr)
	if err != nil {
		log.Fatal(err)
	}
    globalRoom := AddRoom("global")
    go globalRoom.Run()
    fmt.Fprintf(os.Stderr, "LOG [%s] Start listening on %s\n", currentTimeStr(), hostAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
        // This is called Inversion of control ?
        handleNewClient(conn)
	}
}

func currentTimeStr() string {
    return time.Now().Format(time.TimeOnly)
}
