package e2e_test

import "testing"

// TestValidateClean covers `backlog validate` on a well-formed tree (spec §3.8):
// exit 0, json ok=true.
func TestValidateClean(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "good-item", approvedFields("Good item", "cli"), "body")
	writeItem(t, root, "intake", "good-idea", ideaFields("Good idea"), "body")

	r := runIn(t, root, "--json", "backlog", "validate")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["ok"] != true {
		t.Fatalf("clean tree should validate ok=true, got %v\n%s", obj["ok"], r.stdout)
	}
}

// TestValidateIllegalEnum covers exit 3 and a reported violation for an illegal
// enum value (spec §3.8 check 2).
func TestValidateIllegalEnum(t *testing.T) {
	root := mkTree(t)
	f := approvedFields("Bad enum", "cli")
	f["risk"] = "totally-bogus"
	writeItem(t, root, "cli", "bad-enum", f, "body")

	r := runIn(t, root, "--json", "backlog", "validate")
	wantExit(t, r, 3)
	obj := jsonObj(t, r.stdout)
	if obj["ok"] != false {
		t.Fatalf("malformed tree should validate ok=false, got %v", obj["ok"])
	}
	viols, ok := obj["violations"].([]any)
	if !ok || len(viols) == 0 {
		t.Fatalf("expected non-empty violations array, got %v", obj["violations"])
	}
}

// TestValidateWorkstreamMismatch covers exit 3 for an item whose directory does
// not match its workstream field (spec §3.8 check 4).
func TestValidateWorkstreamMismatch(t *testing.T) {
	root := mkTree(t)
	f := approvedFields("Misplaced item", "dev") // says dev, lives in cli/
	writeItem(t, root, "cli", "misplaced-item", f, "body")

	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 3)
}

// TestValidateDuplicateId covers exit 3 for a non-unique id across directories
// (spec §3.8 check 3).
func TestValidateDuplicateId(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "dup-id", approvedFields("Dup one", "cli"), "body")
	writeItem(t, root, "dev", "dup-id", approvedFields("Dup two", "dev"), "body")

	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 3)
}

// TestValidateStrictReadyConsistency covers the --strict-only check 5 (spec
// §3.8): status: approved with an empty/none verify passes default validate but
// fails --strict.
func TestValidateStrictReadyConsistency(t *testing.T) {
	root := mkTree(t)
	f := approvedFields("Strict item", "cli")
	f["verify"] = "none" // approved but not actually ready
	writeItem(t, root, "cli", "strict-item", f, "body")

	// Default validate: this cross-field consistency check is strict-only.
	rDefault := runIn(t, root, "backlog", "validate")
	wantExit(t, rDefault, 0)

	// Strict validate: check 5 fires.
	rStrict := runIn(t, root, "backlog", "validate", "--strict")
	wantExit(t, rStrict, 3)
}

// TestValidateTopLevelAlias covers `anthill validate` as an alias of
// backlog+escalation validate (spec §3 command tree).
func TestValidateTopLevelAlias(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "alias-good", approvedFields("Alias good", "cli"), "body")
	r := runIn(t, root, "validate")
	wantExit(t, r, 0)
}

// TestValidateTopLevelCatchesBadItem confirms the top-level alias also reports
// backlog violations (exit 3).
func TestValidateTopLevelCatchesBadItem(t *testing.T) {
	root := mkTree(t)
	f := approvedFields("Alias bad", "cli")
	f["disposition"] = "not-an-enum"
	writeItem(t, root, "cli", "alias-bad", f, "body")
	r := runIn(t, root, "validate")
	wantExit(t, r, 3)
}
