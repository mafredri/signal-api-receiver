package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/kalbasit/signal-api-receiver/receiver"
)

type mockClient struct {
	connectCalled int
	connectErr    chan error
	recvMsg       chan receiver.Message
	recvErr       chan error
	msgs          []receiver.Message
}

func (mc *mockClient) Connect() error {
	mc.connectCalled++
	if mc.connectErr != nil {
		return <-mc.connectErr
	}
	return nil
}

func (mc *mockClient) ReceiveLoop() error {
	for {
		select {
		case msg := <-mc.recvMsg:
			mc.msgs = append(mc.msgs, msg)
		case err := <-mc.recvErr:
			return err
		}
	}
}

func (mc *mockClient) Pop() *receiver.Message {
	if len(mc.msgs) == 0 {
		return nil
	}

	msg := mc.msgs[0]
	mc.msgs = mc.msgs[1:]

	return &msg
}

func (mc *mockClient) Flush() []receiver.Message {
	msgs := mc.msgs
	mc.msgs = []receiver.Message{}
	return msgs
}

func TestServerReconnect(t *testing.T) {
	mc := &mockClient{
		connectErr: make(chan error),
		recvMsg:    make(chan receiver.Message),
		recvErr:    make(chan error),
	}
	_ = New(mc)

	if want, got := 0, mc.connectCalled; want != got {
		t.Errorf("want %d, got %d", want, got)
	}

	mc.recvMsg <- receiver.Message{Account: "0"}

	mc.recvErr <- nil
	mc.connectErr <- nil

	if want, got := 1, len(mc.msgs); want != got {
		t.Errorf("len(mc.msgs) want %d, got %d", want, got)
	}

	mc.recvMsg <- receiver.Message{Account: "0"}

	if want, got := 1, mc.connectCalled; want != got {
		t.Errorf("mc.connectCalled want %d, got %d", want, got)
	}

	mc.recvErr <- nil

	if want, got := 2, len(mc.msgs); want != got {
		t.Errorf("len(mc.msgs) want %d, got %d", want, got)
	}
}

func TestServeHTTP(t *testing.T) {
	mc := &mockClient{}
	s := New(mc)
	hs := httptest.NewServer(s)
	defer hs.Close()

	t.Run("GET /receive/pop", func(t *testing.T) {
		t.Run("no messages in the queue", func(t *testing.T) {
			mc.msgs = []receiver.Message{}

			resp, err := http.Get(hs.URL + "/receive/pop")
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}
			if want, got := http.StatusNoContent, resp.StatusCode; want != got {
				t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
			}
		})

		t.Run("one message in the queue", func(t *testing.T) {
			want := receiver.Message{Account: "0"}
			mc.msgs = []receiver.Message{want}

			resp, err := http.Get(hs.URL + "/receive/pop")
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}
			if want, got := http.StatusOK, resp.StatusCode; want != got {
				t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			var got receiver.Message
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			if !reflect.DeepEqual(want, got) {
				t.Errorf("want\n%#v\ngot\n%#v", want, got)
			}
		})

		t.Run("three messages in the queue", func(t *testing.T) {
			want := []receiver.Message{
				{Account: "0"},
				{Account: "1"},
				{Account: "2"},
			}
			mc.msgs = want

			var got []receiver.Message

			for range want {
				resp, err := http.Get(hs.URL + "/receive/pop")
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				if want, got := http.StatusOK, resp.StatusCode; want != got {
					t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
				}

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}

				var m receiver.Message
				if err := json.Unmarshal(body, &m); err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				got = append(got, m)
			}

			if !reflect.DeepEqual(want, got) {
				t.Errorf("want\n%#v\ngot\n%#v", want, got)
			}
		})
	})

	t.Run("GET /receive/flush", func(t *testing.T) {
		t.Run("no messages in the queue", func(t *testing.T) {
			mc.msgs = []receiver.Message{}

			resp, err := http.Get(hs.URL + "/receive/flush")
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}
			if want, got := http.StatusOK, resp.StatusCode; want != got {
				t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			var got []receiver.Message
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			if len(got) != 0 {
				t.Errorf("expected an empty response, got\n%#v", got)
			}
		})

		t.Run("one message in the queue", func(t *testing.T) {
			want := []receiver.Message{{Account: "0"}}
			mc.msgs = want

			resp, err := http.Get(hs.URL + "/receive/flush")
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}
			if want, got := http.StatusOK, resp.StatusCode; want != got {
				t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			var got []receiver.Message
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			if !reflect.DeepEqual(want, got) {
				t.Errorf("want\n%#v\ngot\n%#v", want, got)
			}
		})

		t.Run("three messages in the queue", func(t *testing.T) {
			want := []receiver.Message{
				{Account: "0"},
				{Account: "1"},
				{Account: "2"},
			}
			mc.msgs = want

			resp, err := http.Get(hs.URL + "/receive/flush")
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}
			if want, got := http.StatusOK, resp.StatusCode; want != got {
				t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			var got []receiver.Message
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			if !reflect.DeepEqual(want, got) {
				t.Errorf("want\n%#v\ngot\n%#v", want, got)
			}
		})
	})

	t.Run("anything else", func(t *testing.T) {
		mc.msgs = []receiver.Message{}

		for _, verb := range []string{"POST", "PUT", "PATCH", "DELETE"} {
			t.Run(verb+" /", func(t *testing.T) {
				r, err := http.NewRequest(verb, hs.URL, nil)
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				resp, err := http.DefaultClient.Do(r)
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				if want, got := http.StatusForbidden, resp.StatusCode; want != got {
					t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
				}
			})

			t.Run(verb+" /receive/flush", func(t *testing.T) {
				r, err := http.NewRequest(verb, hs.URL+"/receive/flush", nil)
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				resp, err := http.DefaultClient.Do(r)
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				if want, got := http.StatusForbidden, resp.StatusCode; want != got {
					t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
				}
			})

			t.Run(verb+" /receive/pop", func(t *testing.T) {
				r, err := http.NewRequest(verb, hs.URL+"/receive/pop", nil)
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				resp, err := http.DefaultClient.Do(r)
				if err != nil {
					t.Fatalf("expected no error, got: %s", err)
				}
				if want, got := http.StatusForbidden, resp.StatusCode; want != got {
					t.Errorf("want %s, got %s", http.StatusText(want), http.StatusText(got))
				}
			})
		}
	})
}
