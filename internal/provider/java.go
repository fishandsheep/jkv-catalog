package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Java combines the three reviewed Java vendor discovery sources.
type Java struct{}

func (Java) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	var out []Discovery
	for _, source := range []Discoverer{DefaultTemurin(), DefaultDragonwell(), DefaultBiSheng()} {
		found, err := source.Discover(ctx, platform)
		if err != nil {
			return out, err
		}
		out = append(out, found...)
	}
	return out, nil
}

// Temurin discovers supported Java LTS/current majors from Tsinghua's Adoptium mirror.
type Temurin struct {
	BaseURL    string
	HTTPClient *http.Client
}

func DefaultTemurin() Temurin {
	return Temurin{BaseURL: "https://mirrors.tuna.tsinghua.edu.cn/Adoptium/"}
}
func (source Temurin) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.BaseURL == "" {
		source = DefaultTemurin()
	}
	osName := map[string]string{"darwin": "mac", "linux": "linux", "windows": "windows"}[platform.OS]
	if osName == "" {
		return nil, nil
	}
	client := Maven{HTTP: source.HTTPClient}
	re := regexp.MustCompile(`hotspot_([^/]+?)(?:\.tar\.gz|\.zip)$`)
	var out []Discovery
	for _, major := range []string{"8", "11", "17", "21", "25"} {
		base := fmt.Sprintf("%s%s/jdk/%s/%s/", source.BaseURL, major, platform.Arch, osName)
		body, err := client.get(ctx, base)
		if err != nil {
			return nil, err
		}
		for _, link := range links(body) {
			lower := strings.ToLower(link)
			suffix := archiveSuffix(link)
			if !strings.Contains(lower, "openjdk") || !strings.Contains(lower, "-jdk_") || suffix == "" || !archiveFor(platform, suffix) {
				continue
			}
			match := re.FindStringSubmatch(link)
			if len(match) != 2 {
				continue
			}
			version := strings.Replace(match[1], "_", "+", 1)
			out = append(out, Discovery{Candidate: "java", Vendor: "temurin", Version: version, URL: resolve(base, link), ArchiveType: strings.TrimPrefix(suffix, "."), Platform: platform})
		}
	}
	return out, nil
}

// Dragonwell reads Alibaba's structured release metadata.
type Dragonwell struct {
	URL        string
	HTTPClient *http.Client
}

func DefaultDragonwell() Dragonwell {
	return Dragonwell{URL: "https://dragonwell-jdk.io/releases.json"}
}
func (source Dragonwell) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.URL == "" {
		source = DefaultDragonwell()
	}
	if platform.OS == "darwin" {
		return nil, nil
	}
	body, err := (Maven{HTTP: source.HTTPClient}).get(ctx, source.URL)
	if err != nil {
		return nil, err
	}
	var data map[string]map[string]map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	standard := data["oss"]["standard"]
	var out []Discovery
	for _, major := range []string{"8", "11", "17", "21", "25"} {
		version, _ := standard["version"+major].(string)
		key := "xurl" + major
		if platform.Arch == "aarch64" {
			key = "aurl" + major
		} else if platform.OS == "windows" {
			key = "wurl" + major
		}
		raw, _ := standard[key].(string)
		if version != "" && version != "0" && raw != "" {
			out = append(out, Discovery{Candidate: "java", Vendor: "dragonwell", Version: version, URL: raw, ArchiveType: strings.TrimPrefix(archiveSuffix(raw), "."), Platform: platform})
		}
	}
	return out, nil
}

// BiSheng discovers Linux archives from Huawei Cloud's directory index.
type BiSheng struct {
	BaseURL    string
	HTTPClient *http.Client
}

func DefaultBiSheng() BiSheng {
	return BiSheng{BaseURL: "https://mirrors.huaweicloud.com/kunpeng/archive/compiler/bisheng_jdk/"}
}
func (source BiSheng) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.BaseURL == "" {
		source = DefaultBiSheng()
	}
	if platform.OS != "linux" {
		return nil, nil
	}
	body, err := (Maven{HTTP: source.HTTPClient}).get(ctx, source.BaseURL)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`^bisheng-jdk-?((?:8u[0-9]+|(?:11|17|21|25)\.[0-9.]+)(?:-b[0-9]+)?)-linux-` + regexp.QuoteMeta(platform.Arch) + `\.tar\.gz$`)
	var out []Discovery
	for _, link := range links(body) {
		match := re.FindStringSubmatch(link)
		if len(match) != 2 || strings.Contains(link, "debug") || strings.Contains(link, "fusion") {
			continue
		}
		out = append(out, Discovery{Candidate: "java", Vendor: "bisheng", Version: match[1], URL: resolve(source.BaseURL, link), ArchiveType: "tar.gz", Platform: platform})
	}
	return out, nil
}

func archiveSuffix(raw string) string {
	lower := strings.ToLower(raw)
	if strings.HasSuffix(lower, ".tar.gz") {
		return ".tar.gz"
	}
	if strings.HasSuffix(lower, ".tgz") {
		return ".tgz"
	}
	if strings.HasSuffix(lower, ".zip") {
		return ".zip"
	}
	return ""
}
