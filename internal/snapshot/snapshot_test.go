package snapshot

import (
	"bytes"
	"testing"
)

func TestBuildProducesDeterministicValidatedSnapshot(t *testing.T) {
	t.Parallel()

	input := []byte(`{
  "schema_version": 1,
  "sequence": 1,
  "published_at": "2026-07-30T00:00:00Z",
  "source_commit": "0123456789012345678901234567890123456789",
  "min_client_version": "0.3.0-beta.1",
  "candidates": [{
    "name": "maven",
    "display_name": "Apache Maven",
    "description": "Build automation",
    "homepage": "https://maven.apache.org/",
    "default_vendor": "apache",
    "vendors": [{
      "name": "apache",
      "display_name": "Apache",
      "releases": [{
        "version": "3.9.11",
        "selector": "3.9.11",
        "support_tier": "core",
        "artifacts": [{
          "artifact_id": "maven-3.9.11-linux-x64",
          "platforms": [{"os":"linux","arch":"x64"}],
          "archive_type": "tar.gz",
          "url": "https://example.com/maven-3.9.11.tar.gz"
        }]
      }]
    }]
  }]
}`)

	first, err := Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	second, err := Build(input)
	if err != nil {
		t.Fatalf("Build second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("Build output differs for same input")
	}
	if err := Validate(first); err != nil {
		t.Fatalf("Validate output: %v", err)
	}
}
