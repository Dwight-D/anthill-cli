package version

import (
	"strings"
	"testing"
)

func TestStringContainsVersion(t *testing.T) {
	got := String()
	if !strings.Contains(got, Version) {
		t.Fatalf("String() = %q, want it to contain Version %q", got, Version)
	}
}
