// Package provider discovers candidate metadata for maintainer review.
package provider

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

type Platform struct {
	OS   string
	Arch string
}
type Discovery struct {
	Candidate   string
	Vendor      string
	Version     string
	URL         string
	ArchiveType string
	Platform    Platform
}

// FlatArchive discovers release files from a single HTML directory index.
type FlatArchive struct {
	Candidate string
	Vendor    string
	BaseURL   string
	Pattern   string
	HTTP      *http.Client
}

// DefaultFlatArchive returns reviewed discovery coordinates for simple indexes.
func DefaultFlatArchive(candidate string) (FlatArchive, bool) {
	for _, source := range []FlatArchive{
		{Candidate: "gradle", Vendor: "gradle", BaseURL: "https://mirrors.cloud.tencent.com/gradle/", Pattern: `gradle-([0-9]+(?:\.[0-9]+){1,2})-bin`},
		{Candidate: "ant", Vendor: "apache", BaseURL: "https://mirrors.aliyun.com/apache/ant/binaries/", Pattern: `apache-ant-([0-9][0-9A-Za-z.+-]*)-bin`},
		{Candidate: "jmeter", Vendor: "apache", BaseURL: "https://mirrors.aliyun.com/apache/jmeter/binaries/", Pattern: `apache-jmeter-([0-9][0-9A-Za-z.+-]*)`},
	} {
		if source.Candidate == candidate {
			return source, true
		}
	}
	return FlatArchive{}, false
}

var hrefRE = regexp.MustCompile(`(?i)href\s*=\s*["']?([^"' >]+)`)

func (source FlatArchive) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.HTTP == nil {
		source.HTTP = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source.BaseURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "jkv-catalog/0.3")
	response, err := source.HTTP.Do(request)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", source.BaseURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("get %s: HTTP %s", source.BaseURL, response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	pattern, err := regexp.Compile(`^(` + source.Pattern + `)(\.tar\.gz|\.tgz|\.zip)$`)
	if err != nil {
		return nil, fmt.Errorf("compile pattern: %w", err)
	}
	base, _ := url.Parse(source.BaseURL)
	byVersion := map[string]Discovery{}
	for _, found := range hrefRE.FindAllSubmatch(body, -1) {
		name := html.UnescapeString(string(found[1]))
		match := pattern.FindStringSubmatch(name)
		if len(match) != 4 || !stable(match[2]) || !archiveFor(platform, match[3]) {
			continue
		}
		resolved, err := base.Parse(name)
		if err != nil {
			continue
		}
		archive := strings.TrimPrefix(match[3], ".")
		if previous, exists := byVersion[match[2]]; exists && previous.ArchiveType == "tar.gz" {
			continue
		}
		byVersion[match[2]] = Discovery{Candidate: source.Candidate, Vendor: source.Vendor, Version: match[2], URL: resolved.String(), ArchiveType: archive, Platform: platform}
	}
	out := make([]Discovery, 0, len(byVersion))
	for _, release := range byVersion {
		out = append(out, release)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version > out[j].Version })
	return out, nil
}

func stable(version string) bool {
	value := strings.ToUpper(version)
	return !strings.Contains(value, "SNAPSHOT") && !strings.Contains(value, "RC") && !strings.Contains(value, "BETA") && !strings.Contains(value, "ALPHA") && !strings.Contains(value, "MILESTONE") && !strings.Contains(value, "-EA")
}
func archiveFor(platform Platform, suffix string) bool {
	return platform.OS != "windows" || suffix == ".zip"
}
