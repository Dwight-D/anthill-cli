package mdfile

import "testing"

func TestSplitAndCompose(t *testing.T) {
	in := []byte("---\ntitle: hello\nstatus: idea\n---\n\nbody line one\nbody line two\n")
	fm, body, err := Split(in)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if string(fm) != "title: hello\nstatus: idea" {
		t.Fatalf("frontmatter = %q", fm)
	}
	if body != "body line one\nbody line two" {
		t.Fatalf("body = %q", body)
	}
	out := Compose(append(fm, '\n'), body)
	fm2, body2, err := Split(out)
	if err != nil {
		t.Fatalf("re-Split: %v", err)
	}
	if string(fm2) != string(fm) || body2 != body {
		t.Fatalf("round-trip mismatch: fm=%q body=%q", fm2, body2)
	}
}

func TestSplitNoFrontmatter(t *testing.T) {
	if _, _, err := Split([]byte("no fence here\n")); err != ErrNoFrontmatter {
		t.Fatalf("err = %v, want ErrNoFrontmatter", err)
	}
}

func TestSplitCRLF(t *testing.T) {
	in := []byte("---\r\ntitle: x\r\n---\r\nbody\r\n")
	fm, body, err := Split(in)
	if err != nil {
		t.Fatalf("Split CRLF: %v", err)
	}
	if string(fm) != "title: x" || body != "body" {
		t.Fatalf("CRLF handling: fm=%q body=%q", fm, body)
	}
}
