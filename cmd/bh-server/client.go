package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const (
    writeWait = 10 * time.Second
)

type Client struct {
    conn net.Conn

    room *Room

    send chan *Message
}

// Read reads data from Client connection to the room
func (client *Client) Read() {
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
        content = bytes.TrimSpace(bytes.Replace(content, []byte{'\n'}, []byte{' '}, -1)) 
        content = append(content, '\n')
        client.room.broadcast <- &Message{client, content}
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

func (c *Client) register() {
    c.room.register <- c
}

func (c *Client) unregister() {
    c.room.unregister <- c
}

func handleNewClient(c net.Conn) {
	fmt.Fprintf(os.Stderr, "LOG [%s] Received connection: %s\n", time.Now().Format(time.TimeOnly), c.RemoteAddr().String())
    newClient := &Client{conn: c, room: globalRoom, send: make(chan *Message)}
    newClient.register()
    go newClient.Read()
    go newClient.Write()
}

