package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBiShengDiscoverKeepsLatestReleasePerMajor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`
<a href="bisheng-jdk-8u482-b13-linux-x64.tar.gz">old 8</a>
<a href="bisheng-jdk-8u492-b13-linux-x64.tar.gz">latest 8</a>
<a href="bisheng-jdk-11.0.30-b12-linux-x64.tar.gz">old 11</a>
<a href="bisheng-jdk-11.0.31-b12-linux-x64.tar.gz">old build 11</a>
<a href="bisheng-jdk-11.0.31-b13-linux-x64.tar.gz">latest 11</a>
<a href="bisheng-jdk-11.0.31-b13-linux-aarch64.tar.gz">latest 11 arm</a>
<a href="bisheng-jdk-17.0.19-b13-linux-x64.tar.gz">latest 17</a>
<a href="bisheng-jdk-21.0.11-b13-linux-x64.tar.gz">latest 21</a>
<a href="bisheng-jdk-25.0.1-b01-linux-x64.tar.gz">old 25</a>
<a href="bisheng-jdk-25.0.2-b01-linux-aarch64.tar.gz">latest 25, unavailable on x64</a>
<a href="bisheng-jdk-25.0.3-b01-linux-x64-debug.tar.gz">debug</a>`))
	}))
	defer server.Close()

	source := BiSheng{BaseURL: server.URL + "/", HTTPClient: server.Client()}
	got, err := source.Discover(context.Background(), Platform{OS: "linux", Arch: "x64"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []string{"21.0.11-b13", "17.0.19-b13", "11.0.31-b13", "8u492-b13"}
	if len(got) != len(want) {
		t.Fatalf("discoveries = %#v, want versions %v", got, want)
	}
	for index := range want {
		if got[index].Version != want[index] {
			t.Fatalf("discovery %d version = %q, want %q", index, got[index].Version, want[index])
		}
	}
}

func TestDragonwellDiscoverUsesOnlySupportedPlatformURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
  "oss": {
    "standard": {
      "version8": "8.29.28",
      "xurl8": "https://example.com/jdk8-x64-linux.tar.gz",
      "aurl8": "https://example.com/jdk8-aarch64-linux.tar.gz",
      "wurl8": "https://example.com/jdk8-x64-windows.zip"
    }
  }
}`))
	}))
	defer server.Close()

	source := Dragonwell{URL: server.URL, HTTPClient: server.Client()}
	tests := []struct {
		name     string
		platform Platform
		wantURL  string
	}{
		{name: "linux aarch64", platform: Platform{OS: "linux", Arch: "aarch64"}, wantURL: "https://example.com/jdk8-aarch64-linux.tar.gz"},
		{name: "windows x64", platform: Platform{OS: "windows", Arch: "x64"}, wantURL: "https://example.com/jdk8-x64-windows.zip"},
		{name: "windows aarch64 unsupported", platform: Platform{OS: "windows", Arch: "aarch64"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := source.Discover(context.Background(), test.platform)
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			if test.wantURL == "" {
				if len(got) != 0 {
					t.Fatalf("discoveries = %#v, want none", got)
				}
				return
			}
			if len(got) != 1 || got[0].URL != test.wantURL {
				t.Fatalf("discoveries = %#v, want URL %q", got, test.wantURL)
			}
		})
	}
}
