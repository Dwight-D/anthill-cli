package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	cmd := NewRootCommand()

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(--help) returned error: %v", err)
	}

	if got := out.String(); !strings.Contains(got, "anthill") {
		t.Fatalf("help output = %q, want it to contain %q", got, "anthill")
	}
}
