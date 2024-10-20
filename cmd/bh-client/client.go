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
            fmt.Println(string(message)) // NOTE: temp!
        }
    }()

    // Handle WS protocol
    go func() {
        for {
            select {
            case <-done:
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
            case <-interrupt:
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
                case <-done:
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

