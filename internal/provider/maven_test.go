package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMavenDiscovererReadsVersionDirectories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<a href="3.9.11/">ok</a><a href="4.0.0-rc1/">rc</a>`))
		case "/3.9.11/binaries/":
			_, _ = w.Write([]byte(`<a href="apache-maven-3.9.11-bin.tar.gz">tar</a><a href="apache-maven-3.9.11-bin.zip">zip</a>`))
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	got, err := (Maven{RootURL: server.URL + "/", HTTP: server.Client()}).Discover(context.Background(), Platform{OS: "linux", Arch: "x64"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(got) != 1 || got[0].Version != "3.9.11" || !strings.HasSuffix(got[0].URL, ".tar.gz") {
		t.Fatalf("discoveries = %#v", got)
	}
}
