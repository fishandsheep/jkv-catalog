package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Maven discovers Apache Maven distributions from Aliyun's version tree.
type Maven struct {
	RootURL string
	HTTP    *http.Client
}

func DefaultMaven() Maven { return Maven{RootURL: "https://mirrors.aliyun.com/apache/maven/maven-3/"} }

func (source Maven) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.RootURL == "" {
		source = DefaultMaven()
	}
	if source.HTTP == nil {
		source.HTTP = http.DefaultClient
	}
	root, err := source.get(ctx, source.RootURL)
	if err != nil {
		return nil, err
	}
	directoryRE := regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)+/$`)
	versions := []string{}
	for _, link := range links(root) {
		if directoryRE.MatchString(link) && stable(strings.TrimSuffix(link, "/")) {
			versions = append(versions, link)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	var out []Discovery
	for _, version := range versions {
		base := resolve(source.RootURL, version+"binaries/")
		body, err := source.get(ctx, base)
		if err != nil {
			return nil, err
		}
		release, ok := selectArchive("maven", "apache", strings.TrimSuffix(version, "/"), base, body, `apache-maven-([0-9][0-9A-Za-z.+-]*)-bin`, platform)
		if ok {
			out = append(out, release)
		}
	}
	return out, nil
}

func (source Maven) get(ctx context.Context, rawURL string) ([]byte, error) {
	if source.HTTP == nil {
		source.HTTP = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "jkv-catalog/0.3")
	response, err := source.HTTP.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("get %s: HTTP %s", rawURL, response.Status)
	}
	return io.ReadAll(io.LimitReader(response.Body, 16<<20))
}

func links(body []byte) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, found := range hrefRE.FindAllSubmatch(body, -1) {
		link := string(found[1])
		if !seen[link] {
			seen[link] = true
			out = append(out, link)
		}
	}
	return out
}
func resolve(base, ref string) string {
	root, _ := url.Parse(base)
	target, _ := url.Parse(ref)
	return root.ResolveReference(target).String()
}
func selectArchive(candidate, vendor, version, base string, body []byte, pattern string, platform Platform) (Discovery, bool) {
	re := regexp.MustCompile(`^` + pattern + `(\.tar\.gz|\.tgz|\.zip)$`)
	var selected Discovery
	for _, link := range links(body) {
		match := re.FindStringSubmatch(link)
		if len(match) != 3 || match[1] != version || !archiveFor(platform, match[2]) {
			continue
		}
		archive := strings.TrimPrefix(match[2], ".")
		if selected.URL != "" && selected.ArchiveType == "tar.gz" {
			continue
		}
		selected = Discovery{Candidate: candidate, Vendor: vendor, Version: version, URL: resolve(base, link), ArchiveType: archive, Platform: platform}
	}
	return selected, selected.URL != ""
}
