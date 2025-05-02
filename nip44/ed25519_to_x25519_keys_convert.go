package nip44

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"strings"

	"filippo.io/edwards25519"
)

func ConvertEd25519PrivateKeyToX25519(ed25519PrivHex string) (string, error) {
	ed25519PrivHex = strings.TrimPrefix(strings.TrimSpace(ed25519PrivHex), "0x")

	privBytes, err := hex.DecodeString(ed25519PrivHex)
	if err != nil {
		return "", err
	}

	// Check the length of the private key
	// Ed25519 private key has a size of 64 bytes (32 bytes seed + 32 bytes public key)
	// or 32 bytes (only seed)
	var seed []byte
	if len(privBytes) == 64 {
		seed = privBytes[:32]
	} else if len(privBytes) == 32 {
		seed = privBytes
	} else {
		return "", errors.New("invalid ed25519 private key length")
	}

	h := sha512.Sum512(seed)
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64

	return hex.EncodeToString(h[:32]), nil
}

func ConvertEd25519PublicKeyToX25519(ed25519PubHex string) (string, error) {
	ed25519PubHex = strings.TrimPrefix(strings.TrimSpace(ed25519PubHex), "0x")

	pubBytes, err := hex.DecodeString(ed25519PubHex)
	if err != nil {
		return "", err
	}

	if len(pubBytes) != ed25519.PublicKeySize {
		return "", errors.New("invalid ed25519 public key length")
	}

	edPoint, err := edwards25519.NewIdentityPoint().SetBytes(pubBytes)
	if err != nil {
		return "", err
	}

	x25519Bytes := edPoint.BytesMontgomery()

	return hex.EncodeToString(x25519Bytes), nil
}
