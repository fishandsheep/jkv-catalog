package provider

import (
	"context"
	"encoding/xml"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// Groovy discovers binary ZIP distributions from Aliyun's version tree.
type Groovy struct {
	RootURL    string
	HTTPClient *http.Client
}

func DefaultGroovy() Groovy { return Groovy{RootURL: "https://mirrors.aliyun.com/apache/groovy/"} }
func (source Groovy) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.RootURL == "" {
		source = DefaultGroovy()
	}
	client := Maven{HTTP: source.HTTPClient}
	root, err := client.get(ctx, source.RootURL)
	if err != nil {
		return nil, err
	}
	versionRE := regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)+/$`)
	var out []Discovery
	for _, dir := range links(root) {
		version := strings.TrimSuffix(dir, "/")
		if !versionRE.MatchString(dir) || !stable(version) {
			continue
		}
		base := resolve(source.RootURL, dir+"distribution/")
		body, err := client.get(ctx, base)
		if err != nil {
			return nil, err
		}
		name := "apache-groovy-binary-" + version + ".zip"
		if strings.Contains(string(body), name) {
			out = append(out, Discovery{Candidate: "groovy", Vendor: "apache", Version: version, URL: resolve(base, name), ArchiveType: "zip", Platform: platform})
		}
	}
	return out, nil
}

// Tomcat discovers supported major-version binary archives from Aliyun.
type Tomcat struct {
	RootURL    string
	HTTPClient *http.Client
}

func DefaultTomcat() Tomcat { return Tomcat{RootURL: "https://mirrors.aliyun.com/apache/tomcat/"} }
func (source Tomcat) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.RootURL == "" {
		source = DefaultTomcat()
	}
	client := Maven{HTTP: source.HTTPClient}
	root, err := client.get(ctx, source.RootURL)
	if err != nil {
		return nil, err
	}
	branchRE := regexp.MustCompile(`^tomcat-(9|10|11)/$`)
	versionRE := regexp.MustCompile(`^v[0-9]+(?:\.[0-9]+)+/$`)
	var out []Discovery
	for _, branch := range links(root) {
		if !branchRE.MatchString(branch) {
			continue
		}
		branchURL := resolve(source.RootURL, branch)
		body, err := client.get(ctx, branchURL)
		if err != nil {
			return nil, err
		}
		for _, dir := range links(body) {
			if !versionRE.MatchString(dir) {
				continue
			}
			version := strings.Trim(strings.TrimSuffix(dir, "/"), "v")
			base := resolve(branchURL, dir+"bin/")
			b, err := client.get(ctx, base)
			if err != nil {
				return nil, err
			}
			if item, ok := selectArchive("tomcat", "apache", version, base, b, `apache-tomcat-([0-9]+(?:\.[0-9]+)+)`, platform); ok {
				out = append(out, item)
			}
		}
	}
	return out, nil
}

// SpringBoot discovers CLI binary ZIPs from Maven metadata.
type SpringBoot struct {
	RootURL    string
	HTTPClient *http.Client
}

func DefaultSpringBoot() SpringBoot {
	return SpringBoot{RootURL: "https://maven.aliyun.com/repository/central/org/springframework/boot/spring-boot-cli/"}
}
func (source SpringBoot) Discover(ctx context.Context, platform Platform) ([]Discovery, error) {
	if source.RootURL == "" {
		source = DefaultSpringBoot()
	}
	body, err := (Maven{HTTP: source.HTTPClient}).get(ctx, source.RootURL+"maven-metadata.xml")
	if err != nil {
		return nil, err
	}
	var metadata struct {
		Versions []string `xml:"versioning>versions>version"`
	}
	if err := xml.Unmarshal(body, &metadata); err != nil {
		return nil, err
	}
	branchRE := regexp.MustCompile(`^([0-9]+\.[0-9]+)\.`)
	latestByBranch := map[string]string{}
	for _, version := range metadata.Versions {
		match := branchRE.FindStringSubmatch(version)
		if len(match) != 2 || !stable(version) {
			continue
		}
		if current := latestByBranch[match[1]]; current == "" || compareNumericVersions(version, current) > 0 {
			latestByBranch[match[1]] = version
		}
	}
	versions := make([]string, 0, len(latestByBranch))
	for _, version := range latestByBranch {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(left, right int) bool {
		return compareNumericVersions(versions[left], versions[right]) > 0
	})
	var out []Discovery
	for _, version := range versions {
		name := "spring-boot-cli-" + version + "-bin.zip"
		out = append(out, Discovery{Candidate: "springboot", Vendor: "spring", Version: version, URL: source.RootURL + version + "/" + name, ArchiveType: "zip", Platform: platform})
	}
	return out, nil
}
