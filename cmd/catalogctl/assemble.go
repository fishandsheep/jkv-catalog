package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fishandsheep/jkv-catalog/internal/provider"
	"github.com/fishandsheep/jkv-catalog/internal/snapshot"
)

type candidateMeta struct{ name, display, description, homepage, defaultVendor string }

var initialCandidates = []candidateMeta{
	{"java", "Java", "JDK distributions", "https://adoptium.net/", "temurin"},
	{"maven", "Apache Maven", "Build automation", "https://maven.apache.org/", "apache"},
	{"gradle", "Gradle", "Build automation", "https://gradle.org/", "gradle"},
	{"ant", "Apache Ant", "Build automation", "https://ant.apache.org/", "apache"},
	{"groovy", "Apache Groovy", "Programming language", "https://groovy-lang.org/", "apache"},
	{"jmeter", "Apache JMeter", "Load testing", "https://jmeter.apache.org/", "apache"},
	{"tomcat", "Apache Tomcat", "Servlet container", "https://tomcat.apache.org/", "apache"},
	{"springboot", "Spring Boot CLI", "Spring Boot command line tool", "https://spring.io/projects/spring-boot", "spring"},
}

func assemble(ctx context.Context, sequence uint64, publishedAt, sourceCommit string) ([]byte, error) {
	value := snapshot.Snapshot{SchemaVersion: 1, Sequence: sequence, PublishedAt: publishedAt, SourceCommit: sourceCommit, MinClientVersion: "0.3.0-beta.1"}
	platforms := []provider.Platform{{OS: "linux", Arch: "x64"}, {OS: "linux", Arch: "aarch64"}, {OS: "darwin", Arch: "x64"}, {OS: "darwin", Arch: "aarch64"}, {OS: "windows", Arch: "x64"}, {OS: "windows", Arch: "aarch64"}}
	for _, meta := range initialCandidates {
		source, ok := provider.Default(meta.name)
		if !ok {
			return nil, fmt.Errorf("provider missing for %s", meta.name)
		}
		grouped := map[string]map[string][]provider.Discovery{}
		for _, platform := range platforms {
			found, err := source.Discover(ctx, platform)
			if err != nil {
				return nil, fmt.Errorf("discover %s %s/%s: %w", meta.name, platform.OS, platform.Arch, err)
			}
			for _, item := range found {
				if grouped[item.Vendor] == nil {
					grouped[item.Vendor] = map[string][]provider.Discovery{}
				}
				grouped[item.Vendor][item.Version] = append(grouped[item.Vendor][item.Version], item)
			}
		}
		candidate := snapshot.Candidate{Name: meta.name, DisplayName: meta.display, Description: meta.description, Homepage: meta.homepage, DefaultVendor: meta.defaultVendor}
		vendors := sortedMapKeys(grouped)
		for _, vendorName := range vendors {
			vendor := snapshot.Vendor{Name: vendorName, DisplayName: vendorDisplay(vendorName)}
			versions := sortedMapKeys(grouped[vendorName])
			for _, version := range versions {
				artifacts := grouped[vendorName][version]
				release := snapshot.Release{Version: version, Selector: selector(meta.name, vendorName, version), SupportTier: tier(meta.name, vendorName)}
				for _, item := range artifacts {
					release.Artifacts = append(release.Artifacts, snapshot.Artifact{ArtifactID: artifactID(item), ArchiveType: item.ArchiveType, Platforms: []snapshot.Platform{{OS: item.Platform.OS, Arch: item.Platform.Arch}}, URL: item.URL})
				}
				vendor.Releases = append(vendor.Releases, release)
			}
			candidate.Vendors = append(candidate.Vendors, vendor)
		}
		value.Candidates = append(value.Candidates, candidate)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return snapshot.Build(raw)
}

func sortedMapKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys
}
func tier(candidate, vendor string) string {
	if candidate == "maven" || candidate == "gradle" || (candidate == "java" && vendor == "temurin") {
		return "core"
	}
	return "beta"
}
func selector(candidate, vendor, version string) string {
	if candidate == "java" {
		return version + "-" + vendor
	}
	return version
}
func vendorDisplay(vendor string) string {
	if value := map[string]string{"temurin": "Eclipse Temurin", "dragonwell": "Alibaba Dragonwell", "bisheng": "Huawei BiSheng", "apache": "Apache", "gradle": "Gradle", "spring": "Spring"}[vendor]; value != "" {
		return value
	}
	return vendor
}
func artifactID(item provider.Discovery) string {
	value := strings.NewReplacer("+", "-", "/", "-", " ", "-", "_", "-").Replace(item.Version)
	return item.Candidate + "-" + item.Vendor + "-" + value + "-" + item.Platform.OS + "-" + item.Platform.Arch
}

func runAssemble(args []string, stdout, stderr io.Writer) int {
	if len(args) != 5 {
		fmt.Fprintln(stderr, "usage: catalogctl assemble <sequence> <published-at> <source-commit> <output>")
		return 2
	}
	sequence, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil || sequence == 0 {
		fmt.Fprintln(stderr, "assemble: sequence must be positive")
		return 2
	}
	out, err := assemble(context.Background(), sequence, args[2], args[3])
	if err != nil {
		fmt.Fprintf(stderr, "assemble: %v\n", err)
		return 1
	}
	if err := os.WriteFile(args[4], out, 0o644); err != nil {
		fmt.Fprintf(stderr, "write %s: %v\n", args[4], err)
		return 1
	}
	fmt.Fprintf(stdout, "assembled %s\n", args[4])
	return 0
}
