package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const (
    writeWait = 10 * time.Second
)

type Client struct {
    conn net.Conn

    room *Room

    send chan *Message

    username string

    join chan string

    create chan string

    announce chan string
}

func NewClient(c net.Conn) *Client {
    return &Client {
        conn: c,
        room: nil,
        send: make(chan *Message),
        username: "anon", // todo: generate Id
        join: make(chan string),
        create: make(chan string),
        announce: make(chan string),
    }
}

func (c *Client) register() {
    c.room.register <- c
}

func (c *Client) unregister() {
    if (c.room != nil) {
        c.room.unregister <- c
    }
}

func (c *Client) broadcast(content []byte) {
    if (c.room != nil) {
        c.room.broadcast <- &Message{c, content}
    }
}

func (c *Client) switchRoom(newRoom *Room) {
    c.unregister()
    c.room = newRoom
    c.register()
}

func handleNewClient(c net.Conn) {
    fmt.Fprintf(os.Stderr, "LOG [%s] Received connection: %s\n", time.Now().Format(time.TimeOnly), c.RemoteAddr().String())
    newClient := NewClient(c)
    go newClient.handleRequest()
    go newClient.Read()
    go newClient.Write()
}

func (c *Client) handleRequest() {
    for {
        select {
        case username := <- c.announce:
            c.username = username
        case roomName := <- c.create:
            newRoom := AddRoom(roomName)
            newRoom.Run()
            c.switchRoom(newRoom)
        case roomName := <- c.join:
            if room, found := GetRoom(roomName); found {
                c.switchRoom(room)
            }
        }
    }
}

// Read reads data from Client connection to the room
func  (client *Client) Read() {
    var reader *bufio.Reader = bufio.NewReader(client.conn)
    defer func() {
        client.conn.Close()
        client.unregister()
    }()

    for {
        content, err := reader.ReadBytes('\n')
        if err != nil {
            switch err {
            case io.EOF:
                fmt.Fprintf(os.Stdout, "Client [%s] disconnected\n", client.conn.LocalAddr().String())
            default:
                fmt.Fprintf(os.Stdout, "Client [%s] Read error: %s\n", client.conn.LocalAddr().String(), err.Error())
            }
            break;
        }
        fmt.Fprintf(os.Stdout, "[%s]: %s\n", client.conn.RemoteAddr().String(), content)

        // Primitive request dispatch before migration to ws
        // convert only the minimum requirement number of bytes to string
        contentLen := len(content)
        if (contentLen > len("#announce ") && string(content[0:len("#announce ")]) == "#announce ") {
            userName, _ := strings.CutSuffix(string(content[len("#announce "):]), "\n")
            client.announce <- userName
        } else if (contentLen > len("#create ") && string(content[0:len("#create ")]) == "#create ") {
            roomName, _ := strings.CutSuffix(string(content[len("#create "):]), "\n") 
            client.create <- roomName
        } else if (contentLen > len("#join ") && string(content[0:len("#join ")]) == "#join ") {
            roomName, _ := strings.CutSuffix(string(content[len("#join "):]), "\n") 
            client.join <- roomName
        } else {
            client.broadcast(content)
        }
    }
}

// Write writes data from room to Client connection
func (client *Client) Write() {
    for {
        select {
        case message, ok := <-client.send:
            client.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                // the room closed the channel
                client.conn.Write([]byte("Server disconnected"))
                return
            }
            client.conn.Write(message.content)

            // Add queued chat messages to the current websocket message
            n := len(client.send)
            for i := 0; i < n; i++ {
                client.conn.Write([]byte{'\n'})
                message := <-client.send
                client.conn.Write(message.content)
            }
        }
    }
}

