package provider

import "testing"

func TestDefaultFlatArchive(t *testing.T) {
	for _, candidate := range []string{"gradle", "ant", "jmeter"} {
		source, ok := DefaultFlatArchive(candidate)
		if !ok || source.Candidate != candidate || source.BaseURL == "" {
			t.Fatalf("%s = %#v, %v", candidate, source, ok)
		}
	}
	if _, ok := Default("maven"); !ok {
		t.Fatal("maven provider missing")
	}
}
