package cbnet

// WebsocketMessageFrame represents a message struct to communicate web server and client.
type WebsocketMessageFrame struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
