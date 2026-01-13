package nip42

import (
	"net/url"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

// Mock verificator that always returns true (to isolate relay matcher testing)
func mockVerificator(e *nostr.Event) (bool, error) {
	return true, nil
}

func TestValidateAuthEvent_WithCustomRelayMatcher(t *testing.T) {
	t.Parallel()

	challenge := "test-challenge-123"
	relayURL := "wss://relay.example.com"

	createValidEvent := func(eventRelayURL string) *nostr.Event {
		e := CreateUnsignedAuthEvent(challenge, "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", eventRelayURL)
		return &e
	}

	t.Run("default relay matcher - exact match succeeds", func(t *testing.T) {
		evt := createValidEvent("wss://relay.example.com")

		pubkey, err := ValidateAuthEvent(evt, challenge, relayURL, WithCustomVerificator(mockVerificator))
		require.NoError(t, err)
		require.Equal(t, evt.PubKey, pubkey)
	})

	t.Run("default relay matcher - different host fails", func(t *testing.T) {
		evt := createValidEvent("wss://other-relay.example.com")

		_, err := ValidateAuthEvent(evt, challenge, relayURL, WithCustomVerificator(mockVerificator))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid relay URL")
	})

	t.Run("default relay matcher - different scheme fails", func(t *testing.T) {
		evt := createValidEvent("ws://relay.example.com")

		_, err := ValidateAuthEvent(evt, challenge, relayURL, WithCustomVerificator(mockVerificator))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid relay URL")
	})

	t.Run("custom relay matcher - host only matching", func(t *testing.T) {
		evt := createValidEvent("ws://relay.example.com") // Different scheme

		// Custom matcher that only checks host
		hostOnlyMatcher := func(expected, found *url.URL) bool {
			return expected.Host == found.Host
		}

		pubkey, err := ValidateAuthEvent(evt, challenge, relayURL,
			WithCustomVerificator(mockVerificator),
			WithCustomRelayMatcher(hostOnlyMatcher),
		)
		require.NoError(t, err)
		require.Equal(t, evt.PubKey, pubkey)
	})

	t.Run("custom relay matcher - always accept", func(t *testing.T) {
		evt := createValidEvent("wss://completely-different.com")

		// Custom matcher that always accepts
		alwaysAcceptMatcher := func(expected, found *url.URL) bool {
			return true
		}

		pubkey, err := ValidateAuthEvent(evt, challenge, relayURL,
			WithCustomVerificator(mockVerificator),
			WithCustomRelayMatcher(alwaysAcceptMatcher),
		)
		require.NoError(t, err)
		require.Equal(t, evt.PubKey, pubkey)
	})

	t.Run("custom relay matcher - always reject", func(t *testing.T) {
		evt := createValidEvent("wss://relay.example.com") // Exact match

		// Custom matcher that always rejects
		alwaysRejectMatcher := func(expected, found *url.URL) bool {
			return false
		}

		_, err := ValidateAuthEvent(evt, challenge, relayURL,
			WithCustomVerificator(mockVerificator),
			WithCustomRelayMatcher(alwaysRejectMatcher),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid relay URL")
	})

	t.Run("custom relay matcher - subdomain matching", func(t *testing.T) {
		evt := createValidEvent("wss://sub.relay.example.com")
		baseRelayURL := "wss://relay.example.com"

		// Custom matcher that accepts subdomains
		subdomainMatcher := func(expected, found *url.URL) bool {
			return found.Host == expected.Host ||
				len(found.Host) > len(expected.Host) &&
					found.Host[len(found.Host)-len(expected.Host)-1:] == "."+expected.Host
		}

		pubkey, err := ValidateAuthEvent(evt, challenge, baseRelayURL,
			WithCustomVerificator(mockVerificator),
			WithCustomRelayMatcher(subdomainMatcher),
		)
		require.NoError(t, err)
		require.Equal(t, evt.PubKey, pubkey)
	})

	t.Run("custom relay matcher - path prefix matching", func(t *testing.T) {
		evt := createValidEvent("wss://relay.example.com/v1/events")
		relayWithPath := "wss://relay.example.com/v1"

		// Custom matcher that accepts path prefixes
		pathPrefixMatcher := func(expected, found *url.URL) bool {
			if expected.Scheme != found.Scheme || expected.Host != found.Host {
				return false
			}
			return len(found.Path) >= len(expected.Path) &&
				found.Path[:len(expected.Path)] == expected.Path
		}

		pubkey, err := ValidateAuthEvent(evt, challenge, relayWithPath,
			WithCustomVerificator(mockVerificator),
			WithCustomRelayMatcher(pathPrefixMatcher),
		)
		require.NoError(t, err)
		require.Equal(t, evt.PubKey, pubkey)
	})

	t.Run("custom relay matcher receives parsed URLs", func(t *testing.T) {
		evt := createValidEvent("wss://relay.example.com/path")

		var capturedExpected, capturedFound *url.URL
		capturingMatcher := func(expected, found *url.URL) bool {
			capturedExpected = expected
			capturedFound = found
			return true
		}

		_, err := ValidateAuthEvent(evt, challenge, "wss://relay.example.com",
			WithCustomVerificator(mockVerificator),
			WithCustomRelayMatcher(capturingMatcher),
		)
		require.NoError(t, err)

		require.NotNil(t, capturedExpected)
		require.NotNil(t, capturedFound)
		require.Equal(t, "wss", capturedExpected.Scheme)
		require.Equal(t, "relay.example.com", capturedExpected.Host)
		require.Equal(t, "wss", capturedFound.Scheme)
		require.Equal(t, "relay.example.com", capturedFound.Host)
		require.Equal(t, "/path", capturedFound.Path)
	})

	t.Run("custom relay matcher with trailing slash normalization", func(t *testing.T) {
		evt := createValidEvent("wss://relay.example.com/")
		relayWithoutSlash := "wss://relay.example.com"

		// The parseURL function strips trailing slashes, so default matcher should work
		pubkey, err := ValidateAuthEvent(evt, challenge, relayWithoutSlash,
			WithCustomVerificator(mockVerificator),
		)
		require.NoError(t, err)
		require.Equal(t, evt.PubKey, pubkey)
	})

	t.Run("custom relay matcher with case normalization", func(t *testing.T) {
		evt := createValidEvent("WSS://RELAY.EXAMPLE.COM")
		relayLowerCase := "wss://relay.example.com"

		// The parseURL function lowercases URLs, so default matcher should work
		pubkey, err := ValidateAuthEvent(evt, challenge, relayLowerCase,
			WithCustomVerificator(mockVerificator),
		)
		require.NoError(t, err)
		require.Equal(t, evt.PubKey, pubkey)
	})
}

func TestValidateAuthEvent_WithMultipleOptions(t *testing.T) {
	t.Parallel()

	challenge := "test-challenge"
	relayURL := "wss://relay.example.com"

	evt := CreateUnsignedAuthEvent(challenge, `3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d`, "ws://different-relay.com")

	// Use all custom options together
	pubkey, err := ValidateAuthEvent(&evt, challenge, relayURL,
		WithVerificationWindow(1*time.Hour),
		WithCustomVerificator(mockVerificator),
		WithCustomRelayMatcher(func(expected, found *url.URL) bool { return true }),
	)

	require.NoError(t, err)
	require.Equal(t, evt.PubKey, pubkey)
}
