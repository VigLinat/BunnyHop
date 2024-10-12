package main

var globalRoom = newRoom()

type Room struct {
    clients map[*Client]struct{}

    broadcast chan *Message

    register chan *Client

    unregister chan *Client
}

func newRoom() *Room {
    return &Room {
        broadcast:  make(chan *Message),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]struct{}),
    }
}

func (h *Room) run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = struct{}{}
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                if client == message.sender {
                    continue
                }
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
