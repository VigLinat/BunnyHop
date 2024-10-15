package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	ws "github.com/gorilla/websocket"
)

var (
    addr = flag.String("a", "localhost", "Address of server to connect to")
    port = flag.String("p", "50160", "Port of server to connect to")
)

func main() {
    flag.Parse()

    messages := make(chan string)

    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

	remote := fmt.Sprintf("%s:%s", *addr, *port)
    u := url.URL{Scheme: "ws", Host: remote, Path: "/"}
	fmt.Fprintf(os.Stderr, "LOG [%s] Connecting to remote: %s\n", currentTimeStr(), u.String())

    conn, _, err := ws.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatalf("LOG [%s] Dial error: %s\n", currentTimeStr(), err)
    }
	defer func() {
        fmt.Fprintf(os.Stderr, "LOG [%s] Closing connection %s\n", currentTimeStr(), remote)
		conn.Close()
	}()

    done := make(chan struct{})

    // Listen
	// go mustCopy(os.Stdout, conn)
    go func() {
        defer close(done)
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                log.Printf("LOG [%s] Read error: %s\n", currentTimeStr(), err)
                return
            }
            log.Println(message)
        }
    }()

    // Handle WS protocol
    go func() {
        for {
            select {
                case <- done:
                    return
                case message := <- messages:
                    err := conn.WriteMessage(ws.TextMessage, []byte(message))
                    if err != nil {
                        log.Printf("LOG [%s] Write error: %s\n", currentTimeStr(), err)
                        return 
                    }
                case <- interrupt:
                    log.Println("interrupt")

                    // Gracefull shutdown
                    // Close the connection by sending a close message
                    // Then wait (with t/out) for the server to close the connection
                    err := conn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
                    if err != nil {
                        log.Printf("LOG [%s] WriteClose error: %s\n", currentTimeStr(), err)
                        return
                    }
                    select {
                    case <- done:
                    case <-time.After(time.Second):
                    }
                    return
            }
        }
    }()

    // Write
	// mustCopy(conn, os.Stdin)
    input := bufio.NewScanner(os.Stdin)
    for input.Scan() {
        messages <- input.Text()
    }
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}

func currentTimeStr() string {
    return time.Now().Format(time.TimeOnly)
}
