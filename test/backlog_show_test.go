package e2e_test

import "testing"

// TestShowJSON covers `backlog show <id> --json` returning one item object with
// a body string field (spec §3.3).
func TestShowJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "show-item", approvedFields("Show item", "cli"), "The full body text.")
	r := runIn(t, root, "--json", "backlog", "show", "show-item")
	wantExit(t, r, 0)

	obj := jsonObj(t, r.stdout)
	if obj["id"] != "show-item" {
		t.Fatalf("json id = %v, want show-item", obj["id"])
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
	writeItem(t, root, "cli", "nobody-item", approvedFields("Nobody item", "cli"), "Secret body.")
	r := runIn(t, root, "--json", "backlog", "show", "nobody-item", "--no-body")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if body, ok := obj["body"].(string); ok && body != "" {
		t.Fatalf("--no-body should omit/empty the body, got %q", body)
	}
}

// TestShowNotFound covers exit 4 for a missing id.
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
