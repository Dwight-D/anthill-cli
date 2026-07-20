package e2e_test

import (
	"strings"
	"testing"
)

// TestVersionRefJSON covers `anthill version --json` (spec §4.3): the payload
// carries both `version` and a non-empty `template_ref` (the embedded upstream
// template ref an agent records as synced-through on a manual install).
func TestVersionRefJSON(t *testing.T) {
	r := run(t, "--json", "version")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)

	wantHasKey(t, obj, "version")
	if v, _ := obj["version"].(string); strings.TrimSpace(v) == "" {
		t.Fatalf("version --json version empty: %s", r.stdout)
	}
	ref, _ := obj["template_ref"].(string)
	if strings.TrimSpace(ref) == "" {
		t.Fatalf("version --json template_ref missing/empty: %s", r.stdout)
	}
}

// TestVersionRefHuman covers the human form mentioning the embedded template ref
// (spec §4.3: "prints the CLI's own version and the embedded upstream template
// ref").
func TestVersionRefHuman(t *testing.T) {
	ref := templateRef(t)
	r := run(t, "version")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, ref, "version human output")
}
