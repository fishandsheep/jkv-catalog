package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestRunBuildWritesCanonicalSnapshot(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	output := filepath.Join(dir, "catalog-v1.json")
	data := []byte(`{"schema_version":1,"sequence":1,"published_at":"2026-07-30T00:00:00Z","source_commit":"0123456789012345678901234567890123456789","min_client_version":"0.3.0-beta.1","candidates":[{"name":"maven","display_name":"Apache Maven","description":"Build tool","homepage":"https://maven.apache.org/","default_vendor":"apache","vendors":[{"name":"apache","display_name":"Apache","releases":[{"version":"3.9.11","selector":"3.9.11","support_tier":"core","artifacts":[{"artifact_id":"maven-linux-x64","archive_type":"tar.gz","platforms":[{"os":"linux","arch":"x64"}],"url":"https://example.com/maven.tgz"}]}]}]}]}`)
	if err := os.WriteFile(input, data, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"build", input, output}, &stdout, &stderr); code != 0 {
		t.Fatalf("run = %d, stderr = %s", code, stderr.String())
	}
	if code := run([]string{"validate", output}, &stdout, &stderr); code != 0 {
		t.Fatalf("validate = %d, stderr = %s", code, stderr.String())
	}
	if got := stdout.String(); got == "" {
		t.Fatal("expected command output")
	}
}

func TestRunPublicKeyDerivesEd25519PublicKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "private.base64")
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(private)), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"public-key", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("run = %d, stderr = %s", code, stderr.String())
	}
	got, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(stdout.Bytes())))
	if err != nil || !bytes.Equal(got, private.Public().(ed25519.PublicKey)) {
		t.Fatalf("public key = %q, %v", got, err)
	}
}
