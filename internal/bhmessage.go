package internal

type BHMessage struct {
    MsgType string `json:"type"`
    MsgBody []byte `json:"body"`
}
