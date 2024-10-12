package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var addr = flag.String("a", "localhost", "Address of server to connect to")
var port = flag.String("p", "50160", "Port of server to connect to")

func main() {
    flag.Parse()
	remote := fmt.Sprintf("%s:%s", *addr, *port)
	fmt.Fprintf(os.Stderr, "LOG [%s] Connecting to remote: %s\n", currentTimeStr(), remote)
	conn, err := net.Dial("tcp4", remote)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
        fmt.Fprintf(os.Stderr, "LOG [%s] Closing connection %s\n", currentTimeStr(), remote)
		conn.Close()
	}()

	go mustCopy(os.Stdout, conn)
	mustCopy(conn, os.Stdin)
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}

func currentTimeStr() string {
    return time.Now().Format(time.TimeOnly)
}
