package internal

import (
    "os"
    "fmt"
    "time"
)

func MyLog(entry string, v ...any) {
    userLine := fmt.Sprintf(entry, v...)
    fmt.Fprintf(os.Stderr, "LOG [%s]: %s\n", currentTimeStr(), userLine)
}

func currentTimeStr() string {
    return time.Now().Format(time.TimeOnly)
}
