package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSpringBootDiscoverKeepsLatestReleasePerMinor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<metadata><versioning><versions>
<version>3.5.9</version><version>3.5.16</version><version>3.5.17-RC1</version>
<version>4.0.6</version><version>4.0.7</version>
<version>4.1.0-M1</version><version>4.1.0</version>
<version>2.7.17</version><version>2.7.18</version>
</versions></versioning></metadata>`))
	}))
	defer server.Close()

	got, err := (SpringBoot{RootURL: server.URL + "/", HTTPClient: server.Client()}).Discover(context.Background(), Platform{OS: "linux", Arch: "x64"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []string{"4.1.0", "4.0.7", "3.5.16", "2.7.18"}
	if len(got) != len(want) {
		t.Fatalf("discoveries = %#v, want versions %v", got, want)
	}
	for index := range want {
		if got[index].Version != want[index] {
			t.Fatalf("discovery %d version = %q, want %q", index, got[index].Version, want[index])
		}
	}
}
