package nip44

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func HelperGenerateKeyPair(t *testing.T) (private string, public string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err, "failed to generate private key: %v", err)

	return hex.EncodeToString(priv), hex.EncodeToString(pub)
}

func TestGenerateKeys(t *testing.T) {
	privKey, pubKey := HelperGenerateKeyPair(t)
	t.Logf("ED25519 private key: %s", privKey)
	t.Logf("ED25519 public key: %s", pubKey)

	privKeyX25519, err := ConvertEd25519PrivateKeyToX25519(privKey)
	require.NoError(t, err, "failed to convert private key: %v", err)
	t.Logf("X25519 private key: %s", privKeyX25519)

	pubKeyX25519, err := ConvertEd25519PublicKeyToX25519(pubKey)
	require.NoError(t, err, "failed to convert public key: %v", err)
	t.Logf("X25519 public key: %s", pubKeyX25519)

	privKeyX25519Bytes, err := hex.DecodeString(privKeyX25519)
	require.NoError(t, err, "failed to decode X25519 private key: %v", err)
	require.Equal(t, 32, len(privKeyX25519Bytes), "expected X25519 private key length to be 32 bytes, got %d", len(privKeyX25519Bytes))

	pubKeyX25519Bytes, err := hex.DecodeString(pubKeyX25519)
	require.NoError(t, err, "failed to decode X25519 public key: %v", err)
	require.Equal(t, 32, len(pubKeyX25519Bytes), "expected X25519 public key length to be 32 bytes, got %d", len(pubKeyX25519Bytes))
}

func TestConvertEd25519PrivateKeyToX25519(t *testing.T) {
	tests := []struct {
		name           string
		ed25519PrivHex string
		expectedX25519 string
		expectError    bool
	}{
		{
			name:           "Valid conversion",
			ed25519PrivHex: "7da316db30b87b5d4c205bc717ac6f96c8e2e4ffd6bf36b7b3f4fc5ce7eb452c",
			expectedX25519: "282e63367d5f88f29a11be7c86af719a637a97d2b595d72583710554d0b7034d",
			expectError:    false,
		},
		{
			name:           "Invalid hex string",
			ed25519PrivHex: "invalid-hex",
			expectedX25519: "",
			expectError:    true,
		},
		{
			name:           "Invalid key length",
			ed25519PrivHex: "7da316db30b87b5d4c205bc717ac6f96",
			expectedX25519: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x25519Hex, err := ConvertEd25519PrivateKeyToX25519(tt.ed25519PrivHex)
			if (err != nil) != tt.expectError {
				t.Errorf("ConvertEd25519PrivateKeyToX25519() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if err == nil && x25519Hex != tt.expectedX25519 {
				t.Errorf("ConvertEd25519PrivateKeyToX25519() = %v, want %v", x25519Hex, tt.expectedX25519)
			}
		})
	}
}

func TestConvertEd25519PublicKeyToX25519(t *testing.T) {
	tests := []struct {
		name           string
		ed25519PubHex  string
		expectedX25519 string
		expectError    bool
	}{
		{
			name:           "Valid conversion",
			ed25519PubHex:  "1fe71d70cfc69b49fc47406eee7da9722d8d4bed51d21726f079f2d34576b446",
			expectedX25519: "c9cc6f36a8788ac81ea19e1d5982a267a52c9e9083625ca1c3a921ad5488a57b",
			expectError:    false,
		},
		{
			name:           "Invalid hex string",
			ed25519PubHex:  "invalid-hex",
			expectedX25519: "",
			expectError:    true,
		},
		{
			name:           "Invalid key length",
			ed25519PubHex:  "1fe71d70cfc69b49fc47406eee7da9722d8d4bed",
			expectedX25519: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x25519Hex, err := ConvertEd25519PublicKeyToX25519(tt.ed25519PubHex)
			if (err != nil) != tt.expectError {
				t.Errorf("ConvertEd25519PublicKeyToX25519() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if err == nil && x25519Hex != tt.expectedX25519 {
				t.Errorf("ConvertEd25519PublicKeyToX25519() = %v, want %v", x25519Hex, tt.expectedX25519)
			}
		})
	}
}
