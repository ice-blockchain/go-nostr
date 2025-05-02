package nip44

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversationKeyWithExpectedValues(t *testing.T) {
	privKey := "183776f71c9963dc4b767cbb5c0ee28ee91eaca8f9919bc8405295c82f0da61f"
	pubKey := "bf0a14973a578103fb9dcca5225a3545a9893f86dd33d568585d5886db149844"

	conversationKey, err := GenerateConversationKeyX25519(privKey, pubKey)
	require.NoError(t, err, "Error generating conversation key: %v", err)

	plaintext := "Test message for checking the conversation key"
	encrypted, err := EncryptX25519(plaintext, conversationKey, nil)
	require.NoError(t, err, "Error encrypting: %v", err)

	decrypted, err := DecryptX25519(encrypted, conversationKey)
	require.NoError(t, err, "Error decrypting: %v", err)
	require.Equal(t, plaintext, decrypted, "Decrypted message does not match the original")

	t.Logf("Encryption and decryption with conversation key works correctly - OK")
}

func TestConversationKeyInvalid(t *testing.T) {
	tests := []struct {
		name    string
		privKey string
		pubKey  string
		wantErr bool
	}{
		{
			name:    "Valid keys",
			privKey: "183776f71c9963dc4b767cbb5c0ee28ee91eaca8f9919bc8405295c82f0da61f",
			pubKey:  "bf0a14973a578103fb9dcca5225a3545a9893f86dd33d568585d5886db149844",
			wantErr: false,
		},
		{
			name:    "Invalid private key (invalid hex)",
			privKey: "invalid-hex",
			pubKey:  "bf0a14973a578103fb9dcca5225a3545a9893f86dd33d568585d5886db149844",
			wantErr: true,
		},
		{
			name:    "Invalid private key (invalid length)",
			privKey: "123456",
			pubKey:  "bf0a14973a578103fb9dcca5225a3545a9893f86dd33d568585d5886db149844",
			wantErr: true,
		},
		{
			name:    "Invalid public key (invalid hex)",
			privKey: "183776f71c9963dc4b767cbb5c0ee28ee91eaca8f9919bc8405295c82f0da61f",
			pubKey:  "invalid-hex",
			wantErr: true,
		},
		{
			name:    "Invalid public key (invalid length)",
			privKey: "183776f71c9963dc4b767cbb5c0ee28ee91eaca8f9919bc8405295c82f0da61f",
			pubKey:  "123456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateConversationKeyX25519(tt.privKey, tt.pubKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateConversationKeyX25519() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestX25519KeySymmetry(t *testing.T) {
	alicePrivKey := "77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a"
	alicePubKey := "8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a"

	bobPrivKey := "5dab087e624a8a4b79e17f8b83800ee66f3bb1292618b6fd1c2f8b27ff88e0eb"
	bobPubKey := "de9edb7d7b7dc1b4d35b61c2ece435373f8343c85b78674dadfc7e146f882b4f"

	t.Logf("Alice X25519 Private: %s", alicePrivKey)
	t.Logf("Alice X25519 Public:  %s", alicePubKey)
	t.Logf("Bob X25519 Private:   %s", bobPrivKey)
	t.Logf("Bob X25519 Public:    %s", bobPubKey)

	aliceToBobKey, err := GenerateConversationKeyX25519(alicePrivKey, bobPubKey)
	require.NoError(t, err, "Error generating conversation key from Alice to Bob: %v", err)

	bobToAliceKey, err := GenerateConversationKeyX25519(bobPrivKey, alicePubKey)
	require.NoError(t, err, "Error generating conversation key from Bob to Alice: %v", err)

	t.Logf("Conversation key from Alice to Bob: %x", aliceToBobKey)
	t.Logf("Conversation key from Bob to Alice: %x", bobToAliceKey)

	require.True(t, bytes.Equal(aliceToBobKey[:], bobToAliceKey[:]), "Conversation keys do not match for X25519 keys!")
	t.Logf("Conversation keys match for X25519 keys - OK")
	t.Logf("Common conversation key: %x", aliceToBobKey)
}

func TestKeyConversionAndSymmetry(t *testing.T) {
	aliceEd25519PrivKey, aliceEd25519PubKey := HelperGenerateKeyPair(t)
	bobEd25519PrivKey, bobEd25519PubKey := HelperGenerateKeyPair(t)

	fmt.Printf("Alice Ed25519 Private: %s\n", aliceEd25519PrivKey)
	fmt.Printf("Alice Ed25519 Public:  %s\n", aliceEd25519PubKey)
	fmt.Printf("Bob Ed25519 Private:   %s\n", bobEd25519PrivKey)
	fmt.Printf("Bob Ed25519 Public:    %s\n", bobEd25519PubKey)

	aliceX25519PrivKey, err := ConvertEd25519PrivateKeyToX25519(aliceEd25519PrivKey)
	require.NoError(t, err, "Error converting Alice's private key: %v", err)
	aliceX25519PubKey, err := ConvertEd25519PublicKeyToX25519(aliceEd25519PubKey)
	require.NoError(t, err, "Error converting Alice's public key: %v", err)

	bobX25519PrivKey, err := ConvertEd25519PrivateKeyToX25519(bobEd25519PrivKey)
	require.NoError(t, err, "Error converting Bob's private key: %v", err)
	bobX25519PubKey, err := ConvertEd25519PublicKeyToX25519(bobEd25519PubKey)
	require.NoError(t, err, "Error converting Bob's public key: %v", err)

	t.Logf("Alice X25519 Private: %s", aliceX25519PrivKey)
	t.Logf("Alice X25519 Public:  %s", aliceX25519PubKey)
	t.Logf("Bob X25519 Private:   %s", bobX25519PrivKey)
	t.Logf("Bob X25519 Public:    %s", bobX25519PubKey)

	aliceToBobKey, err := GenerateConversationKeyX25519(aliceX25519PrivKey, bobX25519PubKey)
	require.NoError(t, err, "Error generating conversation key from Alice to Bob: %v", err)

	bobToAliceKey, err := GenerateConversationKeyX25519(bobX25519PrivKey, aliceX25519PubKey)
	require.NoError(t, err, "Error generating conversation key from Bob to Alice: %v", err)

	t.Logf("Conversation key from Alice to Bob: %x", aliceToBobKey)
	t.Logf("Conversation key from Bob to Alice: %x", bobToAliceKey)

	require.True(t, bytes.Equal(aliceToBobKey[:], bobToAliceKey[:]), "Conversation keys do not match using our GenerateConversationKeyX25519 function")
	t.Logf("Conversation keys are symmetric when using our GenerateConversationKeyX25519 function - OK")
	t.Logf("Common conversation key: %x", aliceToBobKey)

	testData := "Hello, World!"
	encrypted, err := EncryptX25519(testData, aliceToBobKey, nil)
	require.NoError(t, err, "Error encrypting: %v", err)
	t.Logf("Encrypted message: %s", encrypted)

	decrypted, err := DecryptX25519(encrypted, bobToAliceKey)
	require.NoError(t, err, "Error decrypting: %v", err)
	t.Logf("Decrypted message: %s", decrypted)
}

func TestKnownKeyConversionAndSymmetry(t *testing.T) {
	senderEd25519PubKey := "9e58d6f86dce97b32a86dc7544f21471a4c25506ec9f0c340184b3b6e11980a5"
	receiverEd25519PrivKey := "9a63bfdc132c6f2b1e43f873db02960193bd0ae90eb724a724400b1325905b2c"
	receiverEd25519PubKey := "5cfa36d53eac86221028150a88a3e398d5983b005dcf7b45e447e9d24f80e46c"

	senderX25519PubKey, err := ConvertEd25519PublicKeyToX25519(senderEd25519PubKey)
	require.NoError(t, err, "Error converting Alice's public key: %v", err)

	receiverX25519PrivKey, err := ConvertEd25519PrivateKeyToX25519(receiverEd25519PrivKey)
	require.NoError(t, err, "Error converting Bob's private key: %v", err)

	receiverX25519PubKey, err := ConvertEd25519PublicKeyToX25519(receiverEd25519PubKey)
	require.NoError(t, err, "Error converting Bob's public key: %v", err)

	t.Logf("Alice X25519 Public:  %s", senderX25519PubKey)
	t.Logf("Bob X25519 Private:   %s", receiverX25519PrivKey)
	t.Logf("Bob X25519 Public:    %s", receiverX25519PubKey)

	receiverToSenderKey, err := GenerateConversationKeyX25519(receiverX25519PrivKey, senderX25519PubKey)
	require.NoError(t, err, "Error generating conversation key from Bob to Alice: %v", err)

	require.Equal(t, "03c7306a1299747e29f263c915cc1fe58637bf36adfd537a064cb5252b891ee6", hex.EncodeToString(receiverToSenderKey[:]))

	t.Logf("Conversation key from Bob to Alice: %x", receiverToSenderKey)

	expected := "AhSA7WBdWIvj1SWrn57OP6CU/DRRpH+SycP5tA6RpJKI45hJY5axUzifRyyylD+nJvdz97WqXqiZncgdHOiyiM1Ze+r7tDP+hnl3r8H0QDuzqALGL11qpbrCcTRlcIH4959t"
	decrypted, err := DecryptX25519(expected, receiverToSenderKey)
	require.NoError(t, err)
	require.Equal(t, "hello_world", decrypted)
}
