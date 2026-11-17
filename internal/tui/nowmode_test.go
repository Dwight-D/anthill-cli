package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/session"
)

// nowFixture builds a throwaway schema-3 tree (key prefix ANT):
//
//	cli/0001-working-item.md   ANT-1 in-progress, claimed by cli-worker, session 0123456789abcdef
//	cli/0002-review-item.md    ANT-2 validation, claimed by w2, session fedcba9876543210
//	cli/0003-ready-item.md     ANT-3 triaged, approved, auto + verify (ready)
//	bugs/0004-stuck-item.md    ANT-4 blocked
//	intake/0005-idea-one.md    ANT-5 new, proposed
//
// with ledger records (keyed by item key): ANT-1 last seen long ago; ANT-2
// with no last-seen but a transcript whose mtime is two hours old.
func nowFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".anthill", "backlog")
	for _, d := range []string{"intake", "cli", "bugs"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dir, "workstreams.md"), "---\nschema: 3\nkey-prefix: ANT\nsweep-order: cli, bugs\n---\n")
	write(filepath.Join(dir, "cli/0001-working-item.md"), "---\nworkstream: cli\ntitle: Working item\nsummary: Doing the work\nvalue: v\nstatus: in-progress\ndispatch: approved\ndisposition: auto\nverify: go test\nclaimed-by: cli-worker\nsession: 0123456789abcdef\ncreated-at: \"2026-11-01T00:00:00Z\"\n---\n")
	write(filepath.Join(dir, "cli/0002-review-item.md"), "---\nworkstream: cli\ntitle: Review item\nvalue: v\nstatus: validation\ndispatch: approved\ndisposition: auto\nverify: go test\nclaimed-by: w2\nsession: fedcba9876543210\ndelivered-at: \"2026-11-02T00:00:00Z\"\ncreated-at: \"2026-11-01T00:00:00Z\"\n---\n")
	write(filepath.Join(dir, "cli/0003-ready-item.md"), "---\nworkstream: cli\ntitle: Ready item\nvalue: v\nstatus: triaged\ndispatch: approved\ndisposition: auto\nverify: go test\ncreated-at: \"2026-11-02T00:00:00Z\"\n---\n")
	write(filepath.Join(dir, "bugs/0004-stuck-item.md"), "---\nworkstream: bugs\ntitle: Stuck item\nvalue: v\nstatus: blocked\ndispatch: approved\nclaimed-by: w3\ncreated-at: \"2026-11-01T00:00:00Z\"\n---\n")
	write(filepath.Join(dir, "intake/0005-idea-one.md"), "---\ntitle: Idea one\nvalue: v\nstatus: new\ndispatch: proposed\ncreated-at: \"2026-11-01T00:00:00Z\"\n---\n")

	sdir := filepath.Join(root, ".anthill", "sessions")
	write(filepath.Join(sdir, "0123456789abcdef.json"),
		`{"session":"0123456789abcdef","agent":"","handle":"cli-worker","item":"ANT-1","role":"worker","transcript":"","cwd":"","started":"2020-01-01T00:00:00Z","last-seen":"2020-01-01T00:00:00Z"}`)
	transcript := filepath.Join(root, "transcript.jsonl")
	write(transcript, "{}\n")
	old := time.Now().Add(-2*time.Hour - 5*time.Minute)
	if err := os.Chtimes(transcript, old, old); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(sdir, "fedcba9876543210.json"),
		`{"session":"fedcba9876543210","agent":"","handle":"w2","item":"ANT-2","role":"worker","transcript":`+
			jsonString(transcript)+`,"cwd":"","started":"","last-seen":""}`)
	return root
}

func jsonString(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func loadedNow(t *testing.T, root string) *NowMode {
	t.Helper()
	n := NewNowMode(backlog.NewStore(root), session.NewStore(root))
	if err := n.Reload(); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestNowModeRows(t *testing.T) {
	n := loadedNow(t, nowFixture(t))
	if n.Len() != 2 {
		t.Fatalf("Len = %d, want 2 active items", n.Len())
	}
	ids := map[string]int{n.RowID(0): 0, n.RowID(1): 1}
	wi, ok1 := ids["ANT-1"]
	ri, ok2 := ids["ANT-2"]
	if !ok1 || !ok2 {
		t.Fatalf("rows = %v, want ANT-1 (working) and ANT-2 (review)", ids)
	}
	row := n.Row(wi, 140)
	for _, want := range []string{"ANT-1", "Doing the work", "in-progress", "cli-worker", "01234567", "cli"} {
		if !strings.Contains(row, want) {
			t.Errorf("working row missing %q: %q", want, row)
		}
	}
	if strings.Contains(row, "0123456789") {
		t.Errorf("session must show only its first 8 chars: %q", row)
	}
	if row := n.Row(ri, 140); !strings.Contains(row, "validation") || !strings.Contains(row, "fedcba98") {
		t.Errorf("review row = %q", row)
	}
	// Narrow widths never overflow.
	if got := n.Row(wi, 60); len([]rune(got)) > 60 {
		t.Errorf("row at width 60 is %d cells", len([]rune(got)))
	}

	top, bottom := n.BoardLines(140)
	if len(top) != 1 || !strings.Contains(top[0], "CLAIMED-BY") {
		t.Errorf("top = %q", top)
	}
	all := strings.Join(bottom, "\n")
	for _, want := range []string{"validation", "ANT-2 · Review item", "ready 1", "proposed 1", "blocked 1"} {
		if !strings.Contains(all, want) {
			t.Errorf("board bottom missing %q:\n%s", want, all)
		}
	}
	if strings.Contains(all, "ANT-1 ") || strings.Contains(all, "Working item") {
		t.Errorf("the validation block must list only validation items:\n%s", all)
	}
	if d := n.Detail(wi); !strings.Contains(d, "title:") || !strings.Contains(d, "handle:") {
		t.Errorf("detail lacks item or ledger fields:\n%s", d)
	}
}

func TestNowModeAgeFallback(t *testing.T) {
	root := nowFixture(t)
	n := loadedNow(t, root)
	// Ledger last-seen wins: years old renders in days.
	if age := n.Age("ANT-1"); !strings.HasSuffix(age, "d") {
		t.Errorf("ANT-1 (working) age = %q, want days from last-seen", age)
	}
	// No last-seen: the recorded transcript's mtime (≈2h ago).
	if age := n.Age("ANT-2"); age != "2h" {
		t.Errorf("ANT-2 (review) age = %q, want 2h from the transcript mtime", age)
	}
	// No ledger record at all: "-".
	if err := os.Remove(filepath.Join(root, ".anthill", "sessions", "0123456789abcdef.json")); err != nil {
		t.Fatal(err)
	}
	n = loadedNow(t, root)
	if age := n.Age("ANT-1"); age != "-" {
		t.Errorf("unrecorded age = %q, want -", age)
	}
	for i := 0; i < n.Len(); i++ {
		if n.RowID(i) != "ANT-1" {
			continue
		}
		if _, err := n.transcript(i); err == nil {
			t.Error("transcript action must fail without a recorded path")
		}
	}
}

func TestNowModeTranscriptAndRelease(t *testing.T) {
	root := nowFixture(t)
	n := loadedNow(t, root)
	for i := 0; i < n.Len(); i++ {
		switch n.RowID(i) {
		case "ANT-2": // review-item
			msg, err := n.transcript(i)
			if err != nil || !strings.Contains(msg, "transcript.jsonl") {
				t.Errorf("transcript = %q, %v", msg, err)
			}
		case "ANT-1": // working-item
			if _, err := n.release(i); err != nil {
				t.Fatalf("release: %v", err)
			}
		}
	}
	it, err := backlog.NewStore(root).Find("ANT-1")
	if err != nil || it.Status != backlog.StatusTriaged {
		t.Fatalf("after release: %v %v", it, err)
	}
}

// pollStub is a stubMode that polls.
type pollStub struct{ stubMode }

func (p *pollStub) PollInterval() time.Duration { return time.Millisecond }

func TestTickReloadsPollingModeOnly(t *testing.T) {
	plain := &stubMode{name: "Plain", rows: []string{"a"}}
	poll := &pollStub{stubMode{name: "Poll", rows: []string{"p"}}}
	m := NewWithModes([]Mode{plain, poll})

	// A non-polling first mode schedules no tick.
	if m.pollCmd() != nil {
		t.Fatal("non-polling mode must not schedule a tick")
	}

	// Switching to the polling mode schedules one.
	next, cmd := m.Update(key("2"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("switching to a polling mode must schedule a tick")
	}
	if msg, ok := cmd().(tickMsg); !ok || msg.gen != m.tickGen {
		t.Fatalf("scheduled msg = %#v, want tickMsg gen %d", msg, m.tickGen)
	}

	// A current tick reloads and reschedules.
	before := poll.reloads
	next, cmd = m.Update(tickMsg{gen: m.tickGen})
	m = next.(Model)
	if poll.reloads != before+1 || cmd == nil {
		t.Fatalf("tick: reloads %d→%d, cmd %v", before, poll.reloads, cmd)
	}

	// After switching away, the polling mode's pending tick is stale: no
	// reload of either mode, no reschedule.
	stale := m.tickGen
	next, _ = m.Update(key("1"))
	m = next.(Model)
	pr, ar := poll.reloads, plain.reloads
	next, cmd = m.Update(tickMsg{gen: stale})
	m = next.(Model)
	if cmd != nil || poll.reloads != pr || plain.reloads != ar {
		t.Fatalf("stale tick acted: cmd %v, reloads poll %d→%d plain %d→%d", cmd, pr, poll.reloads, ar, plain.reloads)
	}

	// A tick never reloads under an open confirm prompt, but keeps polling.
	next, _ = m.Update(key("2"))
	m = next.(Model)
	m.confirm = &pendingConfirm{}
	pr = poll.reloads
	_, cmd = m.Update(tickMsg{gen: m.tickGen})
	if poll.reloads != pr || cmd == nil {
		t.Fatalf("tick under confirm: reloads %d→%d, cmd %v", pr, poll.reloads, cmd)
	}
}

func TestNowModeInRoot(t *testing.T) {
	root := nowFixture(t)
	m := New(root)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m = next.(Model)
	next, cmd := m.Update(key("3"))
	m = next.(Model)
	if m.mode().Name() != "Now" || cmd == nil {
		t.Fatalf("[3] → mode %s, tick cmd %v", m.mode().Name(), cmd)
	}
	view := m.View()
	for _, want := range []string{"3:Now", "CLAIMED-BY", "ANT-1", "ready 1"} {
		if !strings.Contains(view, want) {
			t.Errorf("board view missing %q", want)
		}
	}
	// Single pane: the detail viewport spans the full width.
	if m.detailInnerW() != 138 {
		t.Errorf("detailInnerW = %d, want 138", m.detailInnerW())
	}
	// r is the mode's release action (confirm), not the root refresh.
	next, _ = m.Update(key("r"))
	m = next.(Model)
	if m.confirm == nil || m.confirm.action.Label != "release" {
		t.Fatalf("[r] in Now mode must arm release, confirm = %#v", m.confirm)
	}
	next, _ = m.Update(key("n"))
	m = next.(Model)
	// j opens the reject input overlay.
	next, _ = m.Update(key("j"))
	m = next.(Model)
	if m.overlay == nil || m.overlay.act.Label != "reject" {
		t.Fatal("[j] in Now mode must open the reject overlay")
	}
	// enter shows the full-width detail; esc returns to the board.
	m.overlay = nil
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if v := m.View(); !strings.Contains(v, "title:") {
		t.Error("enter must show the selected item's detail")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if !strings.Contains(m.View(), "CLAIMED-BY") {
		t.Error("esc must return to the board")
	}

	// Other modes keep the two-pane layout and the root's r refresh.
	next, _ = m.Update(key("1"))
	m = next.(Model)
	if m.detailInnerW() != m.leftInnerW() {
		t.Error("Backlog mode must keep the two-pane layout")
	}
	next, _ = m.Update(key("r"))
	m = next.(Model)
	if m.status != "refreshed" {
		t.Errorf("[r] in Backlog mode = status %q, want refreshed", m.status)
	}
}
