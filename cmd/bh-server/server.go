package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "time"
    "net/http"
)

var (
    port = flag.String("p","50160", "http server port")
    addr = flag.String("a", "localhost", "http server ip address")
)


func main() {
	flag.Parse()
    hostAddr := fmt.Sprintf("%s:%s", *addr, *port)
    // TODO: delete global room
    globalRoom := AddRoom("global")
    go globalRoom.Run()
    fmt.Fprintf(os.Stderr, "LOG [%s] Start listening on %s\n", currentTimeStr(), hostAddr)

    http.HandleFunc("/", handleNewClient)
    log.Fatal(http.ListenAndServe(hostAddr, nil))
}

func currentTimeStr() string {
    return time.Now().Format(time.TimeOnly)
}
