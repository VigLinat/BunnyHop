package main

var allRooms map[string]*Room = make(map[string]*Room)

type Room struct {
    name string

    clients map[*Client]struct{}

    broadcast chan Message

    register chan *Client

    unregister chan *Client
}

func NewRoom(name string) *Room {
    return &Room{
        name:       name,
        broadcast:  make(chan Message),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]struct{}),
    }
}

func (h *Room) Run() {
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

func GetRoom(name string) (room *Room, found bool) {
    room, found = allRooms[name]
    return
}

func AddRoom(name string) (newRoom *Room) {
    newRoom = NewRoom(name)
    allRooms[name] = newRoom
    return
}
