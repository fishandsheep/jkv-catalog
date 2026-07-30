package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Latest binds an immutable Snapshot asset to its monotonically increasing tag.
type Latest struct {
	SchemaVersion  int    `json:"schema_version"`
	Sequence       uint64 `json:"sequence"`
	SnapshotSHA256 string `json:"snapshot_sha256"`
	ReleaseTag     string `json:"release_tag"`
	SnapshotAsset  string `json:"snapshot_asset"`
}

// BuildLatest emits canonical latest pointer bytes after validating Snapshot.
func BuildLatest(snapshotBytes []byte, asset string) ([]byte, error) {
	var value Snapshot
	if err := decode(snapshotBytes, &value); err != nil {
		return nil, err
	}
	if err := validate(value); err != nil {
		return nil, err
	}
	if asset == "" {
		return nil, fmt.Errorf("snapshot asset is required")
	}
	hash := sha256.Sum256(snapshotBytes)
	latest := Latest{SchemaVersion: 1, Sequence: value.Sequence, SnapshotSHA256: hex.EncodeToString(hash[:]), ReleaseTag: fmt.Sprintf("catalog-v1-%06d", value.Sequence), SnapshotAsset: asset}
	out, err := json.MarshalIndent(latest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
