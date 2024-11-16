package signalapireceiver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Envelope struct {
		Source        string `json:"source"`
		SourceNumber  string `json:"sourceNumber"`
		SourceUUID    string `json:"sourceUuid"`
		SourceName    string `json:"sourceName"`
		SourceDevice  int    `json:"sourceDevice"`
		Timestamp     int64  `json:"timestamp"`
		TypingMessage *struct {
			Action    string `json:"action"`
			Timestamp int64  `json:"timestamp"`
		} `json:"typingMessage,omitempty"`
		DataMessage *struct {
			Timestamp        int64  `json:"timestamp"`
			Message          string `json:"message"`
			ExpiresInSeconds int    `json:"expiresInSeconds"`
			ViewOnce         bool   `json:"viewOnce"`
		} `json:"dataMessage,omitempty"`
	} `json:"envelope"`

	Account string `json:"account"`
}

type Client struct {
	*websocket.Conn

	mu       sync.Mutex
	messages []Message
}

func New(uri *url.URL) (*Client, error) {
	c, _, err := websocket.DefaultDialer.Dial(uri.String(), http.Header{})
	if err != nil {
		return nil, fmt.Errorf("error creating a new websocket connetion: %w", err)
	}

	return &Client{
		Conn:     c,
		messages: []Message{},
	}, nil
}

func (c *Client) ReceiveLoop() {
	log.Print("Starting the receive loop from Signal API")
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("error returned by the websocket: %s", err)
			return
		}
		c.recordMessage(msg)
	}
}

func (c *Client) Flush() []Message {
	c.mu.Lock()
	msgs := c.messages
	c.messages = []Message{}
	c.mu.Unlock()
	return msgs
}

func (c *Client) recordMessage(msg []byte) {
	var m Message
	if err := json.Unmarshal(msg, &m); err != nil {
		log.Printf("error decoding the message below: %s", err)
		log.Print(string(msg[:]))
		return
	}

	// do not record typing messages
	if m.Envelope.DataMessage == nil {
		return
	}

	c.mu.Lock()
	c.messages = append(c.messages, m)
	c.mu.Unlock()

	log.Printf("the following message was successfully recorded: %s", string(msg[:]))
}
