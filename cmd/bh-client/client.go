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
    addr = flag.String("a", "localhost", "Address of server to connect to")
    port = flag.String("p", "50160", "Port of server to connect to")
    messages = make(chan internal.BHMessage)
    remote string
    commandRegex = regexp.MustCompile(`^#(\w+) (\w+)`)
)

func main() {
    flag.Parse()
	remote := fmt.Sprintf("%s:%s", *addr, *port)

    interrupt := make(chan os.Signal, 1)
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

    done := make(chan struct{})

    // Listen
    go func() {
        defer close(done)
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                internal.MyLog("Read error: %s", err)
                return
            }
            fmt.Println(message)
        }
    }()

    // Handle WS protocol
    go func() {
        for {
            select {
                case <- done:
                    return
                case message := <- messages:
                    data, err := json.Marshal(message)
                    if err != nil {
                        internal.MyLog("Marshal error: %s", data)
                    }
                    if err = conn.WriteMessage(ws.TextMessage, data); err != nil {
                        internal.MyLog("Write error: %s", err)
                        return 
                    }
                case <- interrupt:
                    internal.MyLog("SIGINT")

                    // Gracefull shutdown
                    // Close the connection by sending a close message
                    // Then wait (with t/out) for the server to close the connection
                    err := conn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
                    if err != nil {
                        internal.MyLog("WriteClose error: %s", err)
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
    input := bufio.NewScanner(os.Stdin)
    for input.Scan() {
        processUserInput(input.Bytes())
    }

}

func processUserInput(message []byte) {
    bhMessage := internal.BHMessage{}
    if cmd, arg, found := checkIfCommand(message); found {
        bhMessage.MsgType = cmd
        bhMessage.MsgBody = arg
    } else {
        bhMessage.MsgBody = message
        bhMessage.MsgType = "text"
    }
    messages <- bhMessage
}

func checkIfCommand(input []byte) (cmd string, arg []byte, found bool) {
    match := commandRegex.FindSubmatch(input)
    if match == nil || len(match) != 2 {
        return
    }
    cmd, arg, found = string(match[0]), match[1], true
    return
}

