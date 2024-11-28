package nostr

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

const RELAY = "wss://nos.lol"

// test if we can fetch a couple of random events
func TestSubscribeBasic(t *testing.T) {
	t.Parallel()

	rl := mustRelayConnect(t, RELAY)
	defer rl.Close()

	sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{KindTextNote}, Limit: 2}})
	require.NoError(t, err)

	timeout := time.After(5 * time.Second)
	n := 0

	for {
		select {
		case event := <-sub.Events:
			require.NotNil(t, event)
			n++
		case <-sub.EndOfStoredEvents:
			goto end
		case <-rl.Context().Done():
			t.Fatalf("connection closed: %v", rl.Context().Err())
			goto end
		case <-timeout:
			t.Fatalf("timeout")
			goto end
		}
	}

end:
	require.Equal(t, 2, n)
}

// test if we can do multiple nested subscriptions
func TestNestedSubscriptions(t *testing.T) {
	t.Parallel()

	t.Skip("not compatible with the current Nostr version")

	rl := mustRelayConnect(t, RELAY)
	defer rl.Close()

	n := atomic.Uint32{}

	// fetch 2 replies to a note
	sub, err := rl.Subscribe(context.Background(),
		Filters{
			{
				Kinds: []int{KindTextNote},
				Tags:  TagMap{}.SetLiterals("e", "0e34a74f8547e3b95d52a2543719b109fd0312aba144e2ef95cba043f42fe8c5"),
				Limit: 3,
			}})
	require.NoError(t, err)

	for {
		select {
		case event := <-sub.Events:
			// now fetch author of this
			sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{KindProfileMetadata}, Authors: []string{event.PubKey}, Limit: 1}})
			require.NoError(t, err)

			for {
				select {
				case <-sub.Events:
					// do another subscription here in "sync" mode, just so we're sure things are not blocking
					rl.QuerySync(context.Background(), Filter{Limit: 1})

					n.Add(1)
					if n.Load() == 3 {
						// if we get here it means the test passed
						return
					}
				case <-sub.Context.Done():
					goto end
				case <-sub.EndOfStoredEvents:
					sub.Unsub()
				}
			}
		end:
			fmt.Println("")
		case <-sub.EndOfStoredEvents:
			sub.Unsub()
			return
		case <-sub.Context.Done():
			t.Fatalf("connection closed: %v", rl.Context().Err())
			return
		}
	}
}

func TestSubscribeWithExternalSignature(t *testing.T) {
	t.Parallel()

	ch := make(chan struct{})
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		var req ReqEnvelope

		err := websocket.JSON.Receive(conn, &req)
		require.NoError(t, err)

		t.Logf("received subscription request: %v", req)

		var e = EventEnvelope{
			SubscriptionID: &req.SubscriptionID,
			Events: []*Event{
				{
					Kind:    KindTextNote,
					Content: "hello",
				},
			},
		}

		err = websocket.JSON.Send(conn, &e)
		require.NoError(t, err)

		<-ch
	})
	defer ws.Close()

	var didCheck atomic.Bool
	rl := mustRelayConnect(t, ws.URL, WithSignatureChecker(func(event *Event) bool {
		t.Logf("checking signature for event: %v", event)
		didCheck.Store(true)
		return true
	}))

	sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{KindTextNote}, Limit: 1}})
	require.NoError(t, err)

	select {
	case event := <-sub.Events:
		require.NotNil(t, event)
		require.Equal(t, KindTextNote, event.Kind)
		ok, err := event.CheckSignature() // Must be invalid because we didn't sign it.
		require.Error(t, err)
		require.False(t, ok)
		require.True(t, didCheck.Load())
		close(ch)

	case <-sub.EndOfStoredEvents:
		sub.Unsub()

	case <-sub.Context.Done():
		t.Fatalf("connection closed: %v", rl.Context().Err())
	}
}
