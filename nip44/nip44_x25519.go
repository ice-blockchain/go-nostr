package nip44

import (
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const (
	KeySize       = 32
	NonceSize     = 32
	ChaCha20Size  = 32
	ChaCha20Nonce = 12
	HmacSize      = 32
	Base64Version = 2
)

func EncryptX25519(plaintext string, conversationKey [32]byte, customNonce []byte) (string, error) {
	var nonce [32]byte
	if customNonce != nil {
		if len(customNonce) != 32 {
			return "", fmt.Errorf("invalid custom nonce, must be 32 bytes, got %d", len(customNonce))
		}
		copy(nonce[:], customNonce)
	} else {
		if _, err := rand.Read(nonce[:]); err != nil {
			return "", fmt.Errorf("failed to generate random nonce: %w", err)
		}
	}
	chacha20Key, chacha20Nonce, hmacKey, err := messageKeys(conversationKey, nonce)
	if err != nil {
		return "", err
	}

	plain := []byte(plaintext)
	size := len(plain)
	if size < MinPlaintextSize || size > MaxPlaintextSize {
		return "", fmt.Errorf("plaintext should be between 1b and 64kB")
	}

	padding := calcPadding(size)
	padded := make([]byte, 2+padding)
	binary.BigEndian.PutUint16(padded, uint16(size))
	copy(padded[2:], plain)

	ciphertext, err := chacha(chacha20Key, chacha20Nonce, padded)
	if err != nil {
		return "", err
	}

	mac, err := sha256Hmac(hmacKey, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	concat := make([]byte, 1+32+len(ciphertext)+32)
	concat[0] = Base64Version
	copy(concat[1:], nonce[:])
	copy(concat[1+32:], ciphertext)
	copy(concat[1+32+len(ciphertext):], mac)

	return base64.StdEncoding.EncodeToString(concat), nil
}

func DecryptX25519(b64ciphertext string, conversationKey [32]byte) (string, error) {
	cLen := len(b64ciphertext)
	if cLen < 132 || cLen > 87472 {
		return "", fmt.Errorf("invalid payload length: %d", cLen)
	}

	if b64ciphertext[0:1] == "#" {
		return "", fmt.Errorf("unknown version")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid base64")
	}

	if decoded[0] != Base64Version {
		return "", fmt.Errorf("unknown version %d", decoded[0])
	}

	dLen := len(decoded)
	if dLen < 99 || dLen > 65603 {
		return "", fmt.Errorf("invalid data length: %d", dLen)
	}

	var nonce [32]byte
	copy(nonce[:], decoded[1:33])
	ciphertext := decoded[33 : dLen-32]
	givenMac := decoded[dLen-32:]

	chacha20Key, chacha20Nonce, hmacKey, err := messageKeys(conversationKey, nonce)
	if err != nil {
		return "", err
	}

	expectedMac, err := sha256Hmac(hmacKey, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	if !bytesEqual(givenMac, expectedMac) {
		return "", fmt.Errorf("invalid hmac")
	}

	padded, err := chacha(chacha20Key, chacha20Nonce, ciphertext)
	if err != nil {
		return "", err
	}

	if len(padded) < 2 {
		return "", fmt.Errorf("padded data too short")
	}

	unpaddedLen := binary.BigEndian.Uint16(padded[0:2])
	if unpaddedLen < uint16(MinPlaintextSize) || unpaddedLen > uint16(MaxPlaintextSize) ||
		len(padded) != 2+calcPadding(int(unpaddedLen)) {
		return "", fmt.Errorf("invalid padding")
	}

	if len(padded) < 2+int(unpaddedLen) {
		return "", fmt.Errorf("padded data shorter than expected length")
	}

	unpadded := padded[2 : 2+unpaddedLen]
	if len(unpadded) == 0 || len(unpadded) != int(unpaddedLen) {
		return "", fmt.Errorf("invalid padding")
	}

	return string(unpadded), nil
}

func GenerateConversationKeyX25519(privateKeyHex, publicKeyHex string) ([32]byte, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return [32]byte{}, fmt.Errorf("error decoding private key: %w", err)
	}

	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return [32]byte{}, fmt.Errorf("error decoding public key: %w", err)
	}

	if len(privateKeyBytes) != 32 || len(publicKeyBytes) != 32 {
		return [32]byte{}, fmt.Errorf("invalid key length: private key length = %d, public key length = %d", len(privateKeyBytes), len(publicKeyBytes))
	}

	curve := ecdh.X25519()

	privateKey, err := curve.NewPrivateKey(privateKeyBytes)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create private key: %w", err)
	}

	publicKey, err := curve.NewPublicKey(publicKeyBytes)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create public key: %w", err)
	}

	sharedSecret, err := privateKey.ECDH(publicKey)
	if err != nil {
		return [32]byte{}, fmt.Errorf("ECDH failed: %w", err)
	}

	salt := []byte("nip44-v2")
	h := hmac.New(sha256.New, salt)
	h.Write(sharedSecret)

	var conversationKey [32]byte
	copy(conversationKey[:], h.Sum(nil))

	return conversationKey, nil
}

func EncryptWithKeyAndNonce(plaintext string, conversationKeyHex string, nonceHex string) (string, error) {
	var conversationKey [32]byte
	keyBytes, err := hex.DecodeString(conversationKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode conversation key: %w", err)
	}
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("invalid conversation key length: expected 32, got %d", len(keyBytes))
	}
	copy(conversationKey[:], keyBytes)

	nonceBytes, err := hex.DecodeString(nonceHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}
	if len(nonceBytes) != 32 {
		return "", fmt.Errorf("invalid nonce length: expected 32, got %d", len(nonceBytes))
	}

	return EncryptX25519(plaintext, conversationKey, nonceBytes)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
