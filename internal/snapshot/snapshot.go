// Package snapshot builds and validates immutable Catalog v1 payloads.
package snapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var gitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// Snapshot is Catalog v1's signed payload. Field order is deliberately stable.
type Snapshot struct {
	SchemaVersion    int          `json:"schema_version"`
	Sequence         uint64       `json:"sequence"`
	PublishedAt      string       `json:"published_at"`
	SourceCommit     string       `json:"source_commit"`
	MinClientVersion string       `json:"min_client_version"`
	Candidates       []Candidate  `json:"candidates"`
	Revocations      []Revocation `json:"revocations,omitempty"`
}

type Candidate struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Description   string   `json:"description"`
	Homepage      string   `json:"homepage"`
	HomeEnv       string   `json:"home_env,omitempty"`
	DefaultVendor string   `json:"default_vendor"`
	Vendors       []Vendor `json:"vendors"`
}

type Vendor struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Releases    []Release `json:"releases"`
}
type Release struct {
	Version     string     `json:"version"`
	Selector    string     `json:"selector"`
	SupportTier string     `json:"support_tier"`
	Artifacts   []Artifact `json:"artifacts"`
}
type Artifact struct {
	ArtifactID           string     `json:"artifact_id"`
	ArchiveType          string     `json:"archive_type"`
	Platforms            []Platform `json:"platforms"`
	URL                  string     `json:"url"`
	AllowedRedirectHosts []string   `json:"allowed_redirect_hosts,omitempty"`
	Checksum             *Checksum  `json:"checksum,omitempty"`
}
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}
type Checksum struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
	Source    string `json:"source"`
	SourceURL string `json:"source_url,omitempty"`
}
type Revocation struct {
	ArtifactID            string `json:"artifact_id"`
	Reason                string `json:"reason"`
	Message               string `json:"message,omitempty"`
	RevokedAt             string `json:"revoked_at"`
	ReplacementArtifactID string `json:"replacement_artifact_id,omitempty"`
}

// Build validates declarative data then emits canonical, LF-terminated JSON.
func Build(data []byte) ([]byte, error) {
	var value Snapshot
	if err := decode(data, &value); err != nil {
		return nil, err
	}
	if err := validate(value); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	return append(out, '\n'), nil
}

// Validate accepts raw snapshot bytes and verifies protocol constraints.
func Validate(data []byte) error {
	var value Snapshot
	if err := decode(data, &value); err != nil {
		return err
	}
	return validate(value)
}

func decode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid snapshot JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid snapshot JSON: trailing data")
	}
	return nil
}

func validate(value Snapshot) error {
	if value.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema version %d", value.SchemaVersion)
	}
	if value.Sequence == 0 {
		return errors.New("sequence is required")
	}
	if _, err := time.Parse(time.RFC3339, value.PublishedAt); err != nil {
		return errors.New("published_at must be RFC3339")
	}
	if !gitSHA.MatchString(value.SourceCommit) {
		return errors.New("source_commit must be a 40-character Git SHA")
	}
	if value.MinClientVersion == "" {
		return errors.New("min_client_version is required")
	}
	if len(value.Candidates) == 0 {
		return errors.New("candidates is required")
	}
	candidates, selectors, artifacts, revoked := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, candidate := range value.Candidates {
		if candidate.Name == "" || candidate.DisplayName == "" || candidate.Description == "" || candidate.DefaultVendor == "" {
			return errors.New("candidate has required empty field")
		}
		if candidates[candidate.Name] {
			return fmt.Errorf("duplicate candidate %q", candidate.Name)
		}
		candidates[candidate.Name] = true
		if err := https(candidate.Homepage); err != nil {
			return fmt.Errorf("candidate %q homepage: %w", candidate.Name, err)
		}
		vendors, hasDefault, defaultRelease := map[string]bool{}, false, false
		for _, vendor := range candidate.Vendors {
			if vendor.Name == "" || vendor.DisplayName == "" || len(vendor.Releases) == 0 {
				return fmt.Errorf("candidate %q has invalid vendor", candidate.Name)
			}
			if vendors[vendor.Name] {
				return fmt.Errorf("duplicate vendor %q", vendor.Name)
			}
			vendors[vendor.Name] = true
			if vendor.Name == candidate.DefaultVendor {
				hasDefault = true
			}
			versions := map[string]bool{}
			for _, release := range vendor.Releases {
				if release.Version == "" || release.Selector == "" || (release.SupportTier != "core" && release.SupportTier != "beta") || len(release.Artifacts) == 0 {
					return fmt.Errorf("invalid release %q", release.Selector)
				}
				if strings.Contains(strings.ToLower(release.Version), "snapshot") || strings.Contains(strings.ToLower(release.Version), "-rc") {
					return fmt.Errorf("release %q is not stable", release.Version)
				}
				if versions[release.Version] {
					return fmt.Errorf("duplicate release version %q", release.Version)
				}
				versions[release.Version] = true
				selectorKey := candidate.Name + "\x00" + release.Selector
				if selectors[selectorKey] {
					return fmt.Errorf("duplicate release selector %q", release.Selector)
				}
				selectors[selectorKey] = true
				if vendor.Name == candidate.DefaultVendor {
					defaultRelease = true
				}
				platforms := map[string]bool{}
				for _, artifact := range release.Artifacts {
					if artifact.ArtifactID == "" || artifacts[artifact.ArtifactID] {
						return fmt.Errorf("duplicate or empty artifact ID %q", artifact.ArtifactID)
					}
					artifacts[artifact.ArtifactID] = true
					if artifact.ArchiveType != "zip" && artifact.ArchiveType != "tar.gz" && artifact.ArchiveType != "tgz" {
						return fmt.Errorf("unsupported archive type %q", artifact.ArchiveType)
					}
					if err := https(artifact.URL); err != nil {
						return fmt.Errorf("artifact %q URL: %w", artifact.ArtifactID, err)
					}
					for _, platform := range artifact.Platforms {
						key := platform.OS + "/" + platform.Arch
						if platform.OS == "" || platform.Arch == "" || platforms[key] {
							return fmt.Errorf("invalid or ambiguous platform %q", key)
						}
						platforms[key] = true
					}
				}
			}
		}
		if !hasDefault || !defaultRelease {
			return fmt.Errorf("candidate %q default vendor has no release", candidate.Name)
		}
	}
	for _, item := range value.Revocations {
		if item.ArtifactID == "" || item.Reason == "" {
			return errors.New("invalid revocation")
		}
		if revoked[item.ArtifactID] {
			return fmt.Errorf("duplicate revocation %q", item.ArtifactID)
		}
		revoked[item.ArtifactID] = true
		if artifacts[item.ArtifactID] {
			return fmt.Errorf("revoked artifact %q is active", item.ArtifactID)
		}
	}
	return nil
}

func https(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return errors.New("must be an HTTPS URL")
	}
	return nil
}
