// Package signing creates and verifies Catalog v1 detached Ed25519 envelopes.
package signing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type Envelope struct {
	Signatures []Signature `json:"signatures"`
}

type Signature struct {
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"`
}

// Sign returns canonical JSON envelope for payload. Caller owns key storage.
func Sign(payload []byte, keyID string, privateKey ed25519.PrivateKey) ([]byte, error) {
	if keyID == "" {
		return nil, errors.New("key ID is required")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key")
	}
	envelope := Envelope{Signatures: []Signature{{KeyID: keyID, Algorithm: "ed25519", Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))}}}
	out, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode envelope: %w", err)
	}
	return append(out, '\n'), nil
}

// Verify requires at least one signature for a caller-provided trusted key.
func Verify(payload, rawEnvelope []byte, trusted map[string]ed25519.PublicKey) error {
	decoder := json.NewDecoder(bytes.NewReader(rawEnvelope))
	decoder.DisallowUnknownFields()
	var envelope Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("invalid signature envelope: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid signature envelope: trailing data")
	}
	if len(envelope.Signatures) == 0 {
		return errors.New("signature envelope has no signatures")
	}
	seen := map[string]bool{}
	for _, signature := range envelope.Signatures {
		if signature.KeyID == "" || signature.Algorithm != "ed25519" || seen[signature.KeyID] {
			return errors.New("invalid signature envelope")
		}
		seen[signature.KeyID] = true
		public, known := trusted[signature.KeyID]
		if !known {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(signature.Signature)
		if err != nil || len(raw) != ed25519.SignatureSize {
			return errors.New("invalid Ed25519 signature")
		}
		if ed25519.Verify(public, payload, raw) {
			return nil
		}
	}
	return errors.New("no trusted catalog signature verified")
}
