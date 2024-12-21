package receiver

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestFlush(t *testing.T) {
	t.Run("returns empty list when no messages was found", func(t *testing.T) {
		c := &Client{messages: []Message{}}
		if want, got := []Message{}, c.Flush(); !reflect.DeepEqual(want, got) {
			t.Errorf("want %#v got %#v", want, got)
		}
	})

	t.Run("return the message if only one is there", func(t *testing.T) {
		c := &Client{messages: []Message{{Account: "1"}}}

		if want, got := []Message{{Account: "1"}}, c.Flush(); !reflect.DeepEqual(want, got) {
			t.Errorf("want %#v got %#v", want, got)
		}
	})

	t.Run("return messages in order", func(t *testing.T) {
		c := &Client{messages: []Message{
			{Account: "0"},
			{Account: "1"},
			{Account: "2"},
		}}

		want := []Message{
			{Account: "0"},
			{Account: "1"},
			{Account: "2"},
		}
		got := c.Flush()
		if !reflect.DeepEqual(want, got) {
			t.Errorf("want\n%#v\ngot\n%#v", want, got)
		}
	})
}

func TestPop(t *testing.T) {
	t.Run("returns null when no messages was found", func(t *testing.T) {
		c := &Client{messages: []Message{}}
		var want *Message
		if got := c.Pop(); want != got {
			t.Errorf("want %#v got %#v", want, got)
		}
	})

	t.Run("return the message if only one is there", func(t *testing.T) {
		c := &Client{messages: []Message{{Account: "1"}}}

		want := Message{Account: "1"}
		got := c.Pop()
		if !reflect.DeepEqual(want, *got) {
			t.Errorf("want\n%#v\ngot\n%#v", want, got)
		}
	})

	t.Run("return messages in order", func(t *testing.T) {
		c := &Client{messages: []Message{
			{Account: "0"},
			{Account: "1"},
			{Account: "2"},
		}}

		for i := range c.messages {
			want := Message{Account: strconv.Itoa(i)}
			got := c.Pop()
			if !reflect.DeepEqual(want, *got) {
				t.Errorf("want\n%#v\ngot\n%#v", want, got)
			}
		}
	})
}

func TestConnectAndReconnect(t *testing.T) {
	newMessage := func(i int) Message {
		message := strconv.Itoa(i)
		return Message{
			Envelope: Envelope{
				DataMessage: &DataMessage{
					Message: &message,
				},
			},
		}
	}

	ch := make(chan chan Message, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		messages := <-ch
		for msg := range messages {
			if err := conn.WriteJSON(msg); err != nil {
				t.Errorf("write message: %v", err)
				return
			}
		}
	}))
	defer server.Close()

	uri, err := url.Parse("ws" + strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	client, err := New(uri)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for j := 1; j <= 2; j++ {
		messages := make(chan Message, 3)
		for i := 1; i <= cap(messages); i++ {
			messages <- newMessage(j * i)
		}
		close(messages)
		ch <- messages

		err = client.ReceiveLoop()
		if err == nil {
			t.Fatalf("ReceiveLoop: want an error, got nil")
		}

		for i := 1; i <= 3; i++ {
			msg := client.Pop()
			if msg == nil {
				t.Errorf("Pop: want a message, got nil")
			} else if want, got := strconv.Itoa(j*i), *msg.Envelope.DataMessage.Message; want != got {
				t.Errorf("msg.Account: want %q got %q", want, got)
			}
		}

		err = client.Connect()
		if err != nil {
			t.Fatalf("Connect: %v", err)
		}
	}
}
