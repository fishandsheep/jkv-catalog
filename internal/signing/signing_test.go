package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestSignAndVerifyOriginalBytes(t *testing.T) {
	t.Parallel()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("signed bytes\n")
	envelope, err := Sign(payload, "test-key", private)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := Verify(payload, envelope, map[string]ed25519.PublicKey{"test-key": public}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := Verify([]byte("modified bytes\n"), envelope, map[string]ed25519.PublicKey{"test-key": public}); err == nil {
		t.Fatal("Verify accepted modified bytes")
	}
}
