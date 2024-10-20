package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/VigLinat/BunnyHop/internal"
    ws "github.com/gorilla/websocket"
)

const (
    writeWait = 10 * time.Second
    pongWait = 60 * time.Second
    pingPeriod = (pongWait * 9) / 10
    maxMessageSize = 512
)

var upgrader = ws.Upgrader {
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
}

type Client struct {
    conn *ws.Conn
    room *Room
    send chan Message
    username string
    join chan string
    create chan string
}

func NewClient(c *ws.Conn) *Client {
    return &Client{
        conn:     c,
        room:     nil,
        send:     make(chan Message),
        username: "anon", // todo: generate Id
        join:     make(chan string),
        create:   make(chan string),
    }
}

func (c *Client) register() {
    c.room.register <- c
}

func (c *Client) unregister() {
    if c.room != nil {
        c.room.unregister <- c
    }
}

func (c *Client) broadcast(content []byte) {
    if c.room != nil {
        c.room.broadcast <- Message{c, content}
    }
}

func (c *Client) switchRoom(newRoom *Room) {
    c.unregister()
    c.room = newRoom
    c.register()
}

func handleNewClient(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    internal.MyLog("Received connection: %s", conn.RemoteAddr().String())
    newClient := NewClient(conn)
    go newClient.Read()
    go newClient.Write()
}

// Read reads data from Client connection to the room
func (client *Client) Read() {
    defer func() {
        client.unregister()
        client.conn.Close()
    }()
    client.conn.SetReadLimit(maxMessageSize)
    client.conn.SetReadDeadline(time.Now().Add(pongWait))
    client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

    for {
        _, message, err := client.conn.ReadMessage()
        if err != nil {
            fmt.Fprintf(os.Stdout, "Client [%s] Read error: %s\n", client.conn.LocalAddr().String(), err.Error())
            break
        }

        fmt.Fprintf(os.Stdout, "[%s]: %s\n", client.conn.RemoteAddr().String(), message)
        client.handleMessage(message)
    }
}

func (client *Client) handleMessage(message []byte) {
    data := &internal.BHMessage{}
    if err := json.Unmarshal(message, data); err != nil {
        internal.MyLog("Unmarshal error: %s", err)
    }
    switch data.MsgType {
    // TODO: get rid of join and create channels
    case "text":
        client.broadcast(data.MsgBody)
    case "join":
        roomName := string(data.MsgBody)
        if room, found := GetRoom(roomName); found {
            client.switchRoom(room)
        }
    case "create":
        roomName := string(data.MsgBody)
        if _, found := GetRoom(roomName); !found {
            newRoom := AddRoom(roomName)
            go newRoom.Run()
            client.switchRoom(newRoom)
        }
    }

}

// Write writes data from room to Client connection
func (client *Client) Write() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        client.conn.Close()
    }()
    for {
        select {
        case message, ok := <- client.send:
            client.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                client.conn.WriteMessage(ws.CloseMessage, []byte{})
                return
            }

            w, err := client.conn.NextWriter(ws.TextMessage)
            if err != nil {
                return
            }
            w.Write(message.content)

            // Add queued chat messages to the current ws message
            n := len(client.send)
            for i := 0; i < n; i++ {
                w.Write([]byte("\n"))
                message := <- client.send
                w.Write(message.content)
            }

            if err := w.Close(); err != nil {
                return
            }
        case <-ticker.C:
            client.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := client.conn.WriteMessage(ws.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
