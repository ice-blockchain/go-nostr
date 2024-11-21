package nostr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

func TestPublish(t *testing.T) {
	t.Parallel()

	// test note to be sent over websocket
	priv, pub := makeKeyPair(t)
	textNote := Event{
		Kind:      KindTextNote,
		Content:   "hello",
		CreatedAt: Timestamp(1672068534), // random fixed timestamp
		Tags:      Tags{[]string{"foo", "bar"}},
		PubKey:    pub,
	}
	err := textNote.Sign(priv)
	require.NoError(t, err)

	// fake relay server
	var mu sync.Mutex // guards published to satisfy go test -race
	var published bool
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		published = true
		mu.Unlock()
		// verify the client sent exactly the textNote
		var raw []json.RawMessage
		err := websocket.JSON.Receive(conn, &raw)
		require.NoError(t, err)

		event := parseEventMessage(t, raw)
		require.True(t, bytes.Equal(event.Serialize(), textNote.Serialize()))

		// send back an ok nip-20 command result
		res := []any{"OK", textNote.ID, true, ""}
		err = websocket.JSON.Send(conn, res)
		require.NoError(t, err)
	})
	defer ws.Close()

	// connect a client and send the text note
	rl := mustRelayConnect(t, ws.URL)
	err = rl.Publish(context.Background(), textNote)
	require.NoError(t, err)

	require.True(t, published, "fake relay server saw no event")
}

func TestPublishBlocked(t *testing.T) {
	t.Parallel()

	// test note to be sent over websocket
	textNote := Event{Kind: KindTextNote, Content: "hello"}
	textNote.ID = textNote.GetID()

	// fake relay server
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		// discard received message; not interested
		var raw []json.RawMessage
		err := websocket.JSON.Receive(conn, &raw)
		require.NoError(t, err)

		// send back a not ok nip-20 command result
		res := []any{"OK", textNote.ID, false, "blocked"}
		websocket.JSON.Send(conn, res)
	})
	defer ws.Close()

	// connect a client and send a text note
	rl := mustRelayConnect(t, ws.URL)
	err := rl.Publish(context.Background(), textNote)
	require.Error(t, err)
}

func TestPublishWriteFailed(t *testing.T) {
	t.Parallel()

	// test note to be sent over websocket
	textNote := Event{Kind: KindTextNote, Content: "hello"}
	textNote.ID = textNote.GetID()

	// fake relay server
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		// reject receive - force send error
		conn.Close()
	})
	defer ws.Close()

	// connect a client and send a text note
	rl := mustRelayConnect(t, ws.URL)
	// Force brief period of time so that publish always fails on closed socket.
	time.Sleep(1 * time.Millisecond)
	err := rl.Publish(context.Background(), textNote)
	require.Error(t, err)
}

func TestConnectContext(t *testing.T) {
	t.Parallel()

	// fake relay server
	var mu sync.Mutex // guards connected to satisfy go test -race
	var connected bool
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		connected = true
		mu.Unlock()
		io.ReadAll(conn) // discard all input
	})
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := RelayConnect(ctx, ws.URL)
	require.NoError(t, err)

	defer r.Close()

	mu.Lock()
	defer mu.Unlock()
	require.True(t, connected, "fake relay server saw no client connect")
}

func TestConnectContextCanceled(t *testing.T) {
	t.Parallel()

	// fake relay server
	ws := newWebsocketServer(discardingHandler)
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // make ctx expired
	_, err := RelayConnect(ctx, ws.URL)
	require.ErrorIs(t, err, context.Canceled)
}

func TestConnectWithOrigin(t *testing.T) {
	t.Parallel()

	// fake relay server
	// default handler requires origin golang.org/x/net/websocket
	ws := httptest.NewServer(websocket.Handler(discardingHandler))
	defer ws.Close()

	// relay client
	r := NewRelay(context.Background(), NormalizeURL(ws.URL))
	r.RequestHeader = http.Header{"origin": {"https://example.com"}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.Connect(ctx)
	require.NoError(t, err)
}

func discardingHandler(conn *websocket.Conn) {
	io.ReadAll(conn) // discard all input
}

func newWebsocketServer(handler func(*websocket.Conn)) *httptest.Server {
	return httptest.NewServer(&websocket.Server{
		Handshake: anyOriginHandshake,
		Handler:   handler,
	})
}

// anyOriginHandshake is an alternative to default in golang.org/x/net/websocket
// which checks for origin. nostr client sends no origin and it makes no difference
// for the tests here anyway.
var anyOriginHandshake = func(conf *websocket.Config, r *http.Request) error {
	return nil
}

func makeKeyPair(t *testing.T) (priv, pub string) {
	t.Helper()

	privkey := GeneratePrivateKey()
	pubkey, err := GetPublicKey(privkey)
	require.NoError(t, err)

	return privkey, pubkey
}

func mustRelayConnect(t *testing.T, url string) *Relay {
	t.Helper()

	rl, err := RelayConnect(context.Background(), url)
	require.NoError(t, err)

	return rl
}

func parseEventMessage(t *testing.T, raw []json.RawMessage) Event {
	t.Helper()

	require.GreaterOrEqual(t, len(raw), 2)

	var typ string
	err := json.Unmarshal(raw[0], &typ)
	require.NoError(t, err)
	require.Equal(t, "EVENT", typ)

	var event Event
	err = json.Unmarshal(raw[1], &event)
	require.NoError(t, err)

	return event
}

func TestPublishMany(t *testing.T) {
	t.Parallel()

	const (
		nowUnix = 1672068534
		evCount = 42
	)

	var events []*Event
	priv, pub := makeKeyPair(t)

	t.Run("GenerateEvents", func(t *testing.T) {
		for i := range evCount {
			ev := Event{
				Kind:      KindTextNote,
				Content:   "hello " + strconv.Itoa(i),
				CreatedAt: Timestamp(nowUnix + i),
				Tags:      Tags{[]string{"foo", "bar"}},
				PubKey:    pub,
			}
			err := ev.Sign(priv)
			require.NoError(t, err)
			events = append(events, &ev)
		}
		require.Len(t, events, evCount)
	})

	ch := make(chan bool, 1)
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		var e EventEnvelope

		err := websocket.JSON.Receive(conn, &e)
		require.NoError(t, err)

		require.Len(t, e.Events, evCount)
		require.Equal(t, events, e.Events)

		// Send back an ok nip-20 command result.
		for i := range e.Events {
			err := websocket.JSON.Send(conn, &OKEnvelope{EventID: e.Events[i].ID, OK: true})
			require.NoError(t, err)
		}

		ch <- true
	})
	defer ws.Close()

	rl := mustRelayConnect(t, ws.URL)
	require.NoError(t, rl.PublishMany(context.Background()))
	err := rl.PublishMany(context.Background(), events...)
	require.NoError(t, err)

	require.True(t, <-ch, "fake relay server saw no events")
}
