package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFlatArchiveDiscovererFiltersUnstableAndSelectsPlatform(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<a href="gradle-8.14.3-bin.zip">ok</a><a href="gradle-9.0-rc1-bin.zip">rc</a><a href="gradle-8.14.2-bin.zip">old</a>`))
	}))
	defer server.Close()

	discoverer := FlatArchive{Candidate: "gradle", Vendor: "gradle", BaseURL: server.URL + "/", Pattern: `gradle-([0-9]+(?:\.[0-9]+){1,2})-bin`, HTTP: server.Client()}
	got, err := discoverer.Discover(context.Background(), Platform{OS: "linux", Arch: "x64"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(got) != 2 || got[0].Version != "8.14.3" || got[0].ArchiveType != "zip" {
		t.Fatalf("discoveries = %#v", got)
	}
}
