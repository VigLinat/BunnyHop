package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/VigLinat/BunnyHop/internal"
	ws "github.com/gorilla/websocket"
)

var (
    addr         = flag.String("a", "localhost", "Address of server to connect to")
    port         = flag.String("p", "50160", "Port of server to connect to")
    messages     = make(chan internal.BHMessage)
    commandRegex = regexp.MustCompile(`^#(\w+) (\w+)`)
)

var (
    interrupt = make(chan os.Signal, 1) // close connection with Ctrl-C
    done = make(chan struct{}) // remote closed the connection
    closing = make(chan struct{}) // close connection when EOF encountered
)

func main() {
    flag.Parse()
    remote := fmt.Sprintf("%s:%s", *addr, *port)

    signal.Notify(interrupt, os.Interrupt)

    // TODO: make an appropriate endpoint, not '/'
    u := url.URL{Scheme: "ws", Host: remote, Path: "/"}
    internal.MyLog("Connecting to remote: %s", u.String())

    conn, _, err := ws.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        internal.MyLog("Dial error: %s", err)
        os.Exit(1)
    }
    defer func() {
        internal.MyLog("Closing connection %s", remote)
        conn.Close()
    }()

    // Listen
    go func() {
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                internal.MyLog("Read error: %s", err)
                done <- struct{}{}
                return
            }
            fmt.Println(string(message)) // NOTE: temp!
        }
    }()

    // Write
    go func() {
        input := bufio.NewScanner(os.Stdin)
        for {
            ok := input.Scan()
            if !ok {
                err := input.Err()
                // err == nil if input.Scan() encountered io.EOF
                if err != nil {
                    internal.MyLog("Input error: %s", err) 
                }
                closing <- struct{}{}
            }
            processUserInput(input.Bytes())
        }
    }()

    // Handle WS protocol
    for {
        select {
        case <-done:
            return
        case <- closing:
            internal.MyLog("Closing connection to [%s]", conn.RemoteAddr())
            closeConn(conn)
            return
        case <-interrupt:
            internal.MyLog("SIGINT: Closing connection to [%s]", conn.RemoteAddr())
            closeConn(conn)
            return
        case message := <-messages:
            data, err := json.Marshal(message)
            if err != nil {
                internal.MyLog("Marshal error: %s", data)
            }
            if err = conn.WriteMessage(ws.TextMessage, data); err != nil {
                internal.MyLog("Write error: %s", err)
                return
            }
        }
    }
}

// closeConn performs gracefull shutdown
// Close the connection by sending a close message
// Then wait with timeout for the server to close the connection
func closeConn(conn *ws.Conn) {
    err := conn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
    if err != nil {
        internal.MyLog("WriteClose error: %s", err)
        return
    }
    select {
    case <-time.After(time.Second):
    }
}

func processUserInput(message []byte) {
    msgType, msgBody := parseInput(message)
    messages <- internal.BHMessage{MsgType: msgType, MsgBody: msgBody}
}

func parseInput(input []byte) (msgType string, msgBody []byte) {
    match := commandRegex.FindSubmatch(input)
    if match == nil {
        msgType = "text"
        msgBody = input
    } else if len(match) == 3 {
        msgType = string(match[1])
        msgBody = match[2]
    }
    return
}

