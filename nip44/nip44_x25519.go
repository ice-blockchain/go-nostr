package nip44

import (
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	KeySize             = 32
	NonceSize           = 32
	ChaCha20Size        = 32
	PrivateKeySize      = 32
	PublicKeySize       = 32
	SharedSecretSize    = 32
	ConversationKeySize = 32
	HmacSize            = 32
	ChaCha20Nonce       = 12
	MinDataLength       = 99
	MaxDataLength       = 65603
	Base64Version       = 2
	PaddedMinLength     = 2
	DecryptedMinLength  = 132
	DecryptedMaxLength  = 87472
)

func EncryptX25519(plaintext string, conversationKey [ConversationKeySize]byte, customNonce []byte) (string, error) {
	var nonce [NonceSize]byte
	if customNonce != nil {
		if len(customNonce) != NonceSize {
			return "", fmt.Errorf("invalid custom nonce, must be %d bytes, got %d", NonceSize, len(customNonce))
		}
		copy(nonce[:], customNonce)
	} else {
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
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
	padded := make([]byte, PaddedMinLength+padding)
	binary.BigEndian.PutUint16(padded, uint16(size))
	copy(padded[PaddedMinLength:], plain)

	ciphertext, err := chacha(chacha20Key, chacha20Nonce, padded)
	if err != nil {
		return "", err
	}

	mac, err := sha256Hmac(hmacKey, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	concat := make([]byte, 1+NonceSize+len(ciphertext)+HmacSize)
	concat[0] = Base64Version
	copy(concat[1:], nonce[:])
	copy(concat[1+NonceSize:], ciphertext)
	copy(concat[1+NonceSize+len(ciphertext):], mac)

	return base64.StdEncoding.EncodeToString(concat), nil
}

func DecryptX25519(b64ciphertext string, conversationKey [ConversationKeySize]byte) (string, error) {
	cLen := len(b64ciphertext)
	if cLen < DecryptedMinLength || cLen > DecryptedMaxLength {
		return "", fmt.Errorf("invalid payload length: %d", cLen)
	}

	if b64ciphertext[0:1] == "#" {
		return "", fmt.Errorf("unknown version")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	if decoded[0] != Base64Version {
		return "", fmt.Errorf("unknown version %d, expected %d", decoded[0], Base64Version)
	}

	dLen := len(decoded)
	if dLen < MinDataLength || dLen > MaxDataLength {
		return "", fmt.Errorf("invalid data length: %d, min %d, max %d", dLen, MinDataLength, MaxDataLength)
	}

	var nonce [NonceSize]byte
	copy(nonce[:], decoded[1:1+NonceSize])
	ciphertext := decoded[1+NonceSize : dLen-HmacSize]
	givenMac := decoded[dLen-HmacSize:]

	chacha20Key, chacha20Nonce, hmacKey, err := messageKeys(conversationKey, nonce)
	if err != nil {
		return "", err
	}

	expectedMac, err := sha256Hmac(hmacKey, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	if subtle.ConstantTimeCompare(givenMac, expectedMac) != 1 {
		return "", fmt.Errorf("invalid hmac")
	}

	padded, err := chacha(chacha20Key, chacha20Nonce, ciphertext)
	if err != nil {
		return "", err
	}

	if len(padded) < PaddedMinLength {
		return "", fmt.Errorf("padded data too short")
	}

	unpaddedLen := binary.BigEndian.Uint16(padded[0:PaddedMinLength])
	if unpaddedLen < uint16(MinPlaintextSize) || unpaddedLen > uint16(MaxPlaintextSize) ||
		len(padded) != PaddedMinLength+calcPadding(int(unpaddedLen)) {
		return "", fmt.Errorf("invalid padding")
	}

	if len(padded) < PaddedMinLength+int(unpaddedLen) {
		return "", fmt.Errorf("padded data shorter than expected length")
	}

	unpadded := padded[PaddedMinLength : PaddedMinLength+unpaddedLen]
	if len(unpadded) == 0 || len(unpadded) != int(unpaddedLen) {
		return "", fmt.Errorf("invalid padding")
	}

	return string(unpadded), nil
}

func GenerateConversationKeyX25519(privateKeyHex, publicKeyHex string) ([ConversationKeySize]byte, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return [PrivateKeySize]byte{}, fmt.Errorf("error decoding private key: %w", err)
	}

	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return [PublicKeySize]byte{}, fmt.Errorf("error decoding public key: %w", err)
	}

	if len(privateKeyBytes) != PrivateKeySize || len(publicKeyBytes) != PublicKeySize {
		return [PrivateKeySize]byte{}, fmt.Errorf("invalid key length: private key length = %d, public key length = %d", len(privateKeyBytes), len(publicKeyBytes))
	}

	curve := ecdh.X25519()

	privateKey, err := curve.NewPrivateKey(privateKeyBytes)
	if err != nil {
		return [PrivateKeySize]byte{}, fmt.Errorf("failed to create private key: %w", err)
	}

	publicKey, err := curve.NewPublicKey(publicKeyBytes)
	if err != nil {
		return [PublicKeySize]byte{}, fmt.Errorf("failed to create public key: %w", err)
	}

	sharedSecret, err := privateKey.ECDH(publicKey)
	if err != nil {
		return [SharedSecretSize]byte{}, fmt.Errorf("ECDH failed: %w", err)
	}

	salt := []byte("nip44-v2")
	h := hmac.New(sha256.New, salt)
	h.Write(sharedSecret)

	var conversationKey [ConversationKeySize]byte
	copy(conversationKey[:], h.Sum(nil))

	return conversationKey, nil
}

func EncryptWithKeyAndNonce(plaintext string, conversationKeyHex string, nonceHex string) (string, error) {
	var conversationKey [ConversationKeySize]byte
	keyBytes, err := hex.DecodeString(conversationKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode conversation key: %w", err)
	}
	if len(keyBytes) != ConversationKeySize {
		return "", fmt.Errorf("invalid conversation key length: expected %d, got %d", ConversationKeySize, len(keyBytes))
	}
	copy(conversationKey[:], keyBytes)

	nonceBytes, err := hex.DecodeString(nonceHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}
	if len(nonceBytes) != NonceSize {
		return "", fmt.Errorf("invalid nonce length: expected %d, got %d", NonceSize, len(nonceBytes))
	}

	return EncryptX25519(plaintext, conversationKey, nonceBytes)
}
