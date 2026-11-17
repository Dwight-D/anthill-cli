package e2e_test

import "testing"

// TestShowJSON covers `backlog show <id> --json` returning one item object with
// a body string field (spec §3.3). The JSON id is the canonical key.
func TestShowJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "show-item", readyFields("Show item", "cli"), "The full body text.")
	r := runIn(t, root, "--json", "backlog", "show", key(1))
	wantExit(t, r, 0)

	obj := jsonObj(t, r.stdout)
	if obj["id"] != key(1) {
		t.Fatalf("json id = %v, want %s", obj["id"], key(1))
	}
	body, ok := obj["body"].(string)
	if !ok {
		t.Fatalf("show --json missing string body field: %v", obj)
	}
	wantContains(t, body, "The full body text.", "show body")
}

// TestShowNoBody covers --no-body suppressing the markdown body.
func TestShowNoBody(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "nobody-item", readyFields("Nobody item", "cli"), "Secret body.")
	r := runIn(t, root, "--json", "backlog", "show", key(1), "--no-body")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if body, ok := obj["body"].(string); ok && body != "" {
		t.Fatalf("--no-body should omit/empty the body, got %q", body)
	}
}

// TestShowNotFound covers exit 4 for a missing id. The id is slug-shaped (a
// former-id lookup that matches nothing); a malformed id is exit 2 instead.
func TestShowNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "show", "ghost-item")
	wantExit(t, r, 4)
}

// TestShowNotFoundJSONError covers the structured error envelope on stderr under
// --json (spec §2 JSON envelope): { "error": { "code", "exit", "message", ... } }.
func TestShowNotFoundJSONError(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "--json", "backlog", "show", "ghost-item")
	wantExit(t, r, 4)

	env := jsonObj(t, r.stderr)
	errObj, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected an error envelope object on stderr, got: %q", r.stderr)
	}
	if errObj["code"] != "not_found" {
		t.Fatalf("error.code = %v, want not_found", errObj["code"])
	}
	if n, ok := errObj["exit"].(float64); !ok || int(n) != 4 {
		t.Fatalf("error.exit = %v, want 4", errObj["exit"])
	}
	if _, ok := errObj["message"].(string); !ok {
		t.Fatalf("error envelope missing message: %v", errObj)
	}
}
