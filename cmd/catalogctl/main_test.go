package main

import (
	"bytes"
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
