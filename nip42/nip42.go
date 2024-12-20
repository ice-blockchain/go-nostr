package nip42

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// CreateUnsignedAuthEvent creates an event which should be sent via an "AUTH" command.
// If the authentication succeeds, the user will be authenticated as pubkey.
func CreateUnsignedAuthEvent(challenge, pubkey, relayURL string) nostr.Event {
	return nostr.Event{
		PubKey:    pubkey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindClientAuthentication,
		Tags: nostr.Tags{
			nostr.Tag{"relay", relayURL},
			nostr.Tag{"challenge", challenge},
		},
		Content: "",
	}
}

// helper function for ValidateAuthEvent.
func parseURL(input string) (*url.URL, error) {
	return url.Parse(
		strings.ToLower(
			strings.TrimSuffix(input, "/"),
		),
	)
}

type options struct {
	VerificationWindow time.Duration
	Verificator        func(*nostr.Event) (bool, error)
}

// Option is a function that modifies the options.
type Option func(*options)

// WithVerificationWindow sets the time window for the event verification.
func WithVerificationWindow(window time.Duration) func(*options) {
	return func(o *options) {
		o.VerificationWindow = window
	}
}

// WithCustomVerificator sets the custom event verification function.
func WithCustomVerificator(verificator func(*nostr.Event) (bool, error)) func(*options) {
	return func(o *options) {
		o.Verificator = verificator
	}
}

// ValidateAuthEvent checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func ValidateAuthEvent(event *nostr.Event, challenge string, relayURL string, opts ...Option) (pubkey string, err error) {
	var options = options{
		VerificationWindow: 10 * time.Minute,
		Verificator:        func(e *nostr.Event) (bool, error) { return e.CheckSignature() },
	}

	for _, opt := range opts {
		opt(&options)
	}

	if event.Kind != nostr.KindClientAuthentication {
		return "", fmt.Errorf("invalid event kind: %v, expected %v", event.Kind, nostr.KindClientAuthentication)
	}

	if event.Tags.GetFirst([]string{"challenge", challenge}) == nil {
		return "", fmt.Errorf("missing or invalid challenge tag")
	}

	expected, err := parseURL(relayURL)
	if err != nil {
		return "", fmt.Errorf("cannot parse input relayURL: %w", err)
	}

	found, err := parseURL(event.Tags.GetFirst([]string{"relay", ""}).Value())
	if err != nil {
		return "", fmt.Errorf("cannot parse event relay URL: %w", err)
	}

	if expected.Scheme != found.Scheme ||
		expected.Host != found.Host ||
		expected.Path != found.Path {
		return "", fmt.Errorf("invalid relay URL: %q, expected %q", found.String(), expected.String())
	}

	now := time.Now()
	if event.CreatedAt.Time().After(now.Add(options.VerificationWindow)) || event.CreatedAt.Time().Before(now.Add(-options.VerificationWindow)) {
		return "", fmt.Errorf("event is too old or too new: %v", event.CreatedAt)
	}

	// save for last, as it is most expensive operation
	// no need to check returned error, since ok == true implies err == nil.
	if ok, err := options.Verificator(event); err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	} else if !ok {
		return "", fmt.Errorf("invalid signature")
	}

	return event.PubKey, nil
}
