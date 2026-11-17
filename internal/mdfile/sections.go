package mdfile

import "strings"

// Section is one "## Heading" block of a markdown body. The preamble (any text
// before the first "## " heading) is a Section with an empty Heading.
type Section struct {
	Heading string
	Body    string
}

// isFence reports whether line opens or closes a fenced code block.
func isFence(line string) bool {
	t := strings.TrimLeft(line, " ")
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// headingOf returns the heading text of a level-2 heading line ("## X" -> "X").
func headingOf(line string) (string, bool) {
	if !strings.HasPrefix(line, "## ") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "## ")), true
}

// ParseSections splits a markdown body into its ordered "## " sections. Lines
// inside fenced code blocks are never treated as headings. Section bodies are
// whitespace-trimmed; unknown headings are kept in order. An empty preamble is
// omitted.
func ParseSections(body string) []Section {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	var out []Section
	cur := Section{}
	started := false
	var buf []string
	inFence := false
	flush := func() {
		cur.Body = strings.TrimSpace(strings.Join(buf, "\n"))
		if started || cur.Body != "" {
			out = append(out, cur)
		}
		buf = nil
	}
	for _, line := range strings.Split(body, "\n") {
		if isFence(line) {
			inFence = !inFence
		}
		if !inFence {
			if h, ok := headingOf(line); ok {
				flush()
				cur = Section{Heading: h}
				started = true
				continue
			}
		}
		buf = append(buf, line)
	}
	flush()
	return out
}

// SectionMap returns heading -> trimmed body for every section of body. When a
// heading repeats, the first occurrence wins. The preamble is not included.
func SectionMap(body string) map[string]string {
	m := map[string]string{}
	for _, s := range ParseSections(body) {
		if s.Heading == "" {
			continue
		}
		if _, dup := m[s.Heading]; !dup {
			m[s.Heading] = s.Body
		}
	}
	return m
}

// RenderSections renders sections into a markdown body: each section is its
// heading line, its body (when non-empty), and a blank separator line.
func RenderSections(secs []Section) string {
	var b strings.Builder
	for i, s := range secs {
		if i > 0 {
			b.WriteString("\n")
		}
		if s.Heading != "" {
			b.WriteString("## " + s.Heading + "\n")
		}
		if s.Body != "" {
			b.WriteString(s.Body + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// AppendToSection appends text to the end of the named section's body, editing
// the body textually so every other byte of prose is preserved. When the
// section is absent it is created: placed after the last present heading that
// precedes it in order (order is the canonical heading sequence), before the
// first present heading that follows it, or at the end of the body.
func AppendToSection(body, heading, text string, order []string) string {
	body = strings.TrimRight(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	text = strings.TrimRight(text, "\n")
	lines := []string{}
	if body != "" {
		lines = strings.Split(body, "\n")
	}
	type hpos struct {
		name string
		idx  int
	}
	var heads []hpos
	inFence := false
	for i, l := range lines {
		if isFence(l) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if h, ok := headingOf(l); ok {
			heads = append(heads, hpos{h, i})
		}
	}
	endOf := func(k int) int { // exclusive end line index of heads[k]'s section
		if k+1 < len(heads) {
			return heads[k+1].idx
		}
		return len(lines)
	}
	for k, h := range heads {
		if h.name != heading {
			continue
		}
		end := endOf(k)
		ins := end
		for ins > h.idx+1 && strings.TrimSpace(lines[ins-1]) == "" {
			ins--
		}
		block := strings.Split(text, "\n")
		if end < len(lines) {
			block = append(block, "")
		}
		out := append([]string{}, lines[:ins]...)
		out = append(out, block...)
		out = append(out, lines[end:]...)
		return strings.Join(out, "\n")
	}
	rank := map[string]int{}
	for i, n := range order {
		rank[n] = i
	}
	insertAt := len(lines)
	if want, ok := rank[heading]; ok {
		best := -1
		for k, h := range heads {
			if r, ok := rank[h.name]; ok && r < want {
				best = k
			}
		}
		if best >= 0 {
			insertAt = endOf(best)
		} else {
			for _, h := range heads {
				if r, ok := rank[h.name]; ok && r > want {
					insertAt = h.idx
					break
				}
			}
		}
	}
	newSec := []string{"## " + heading}
	if text != "" {
		newSec = append(newSec, strings.Split(text, "\n")...)
	}
	pre := append([]string{}, lines[:insertAt]...)
	for len(pre) > 0 && strings.TrimSpace(pre[len(pre)-1]) == "" {
		pre = pre[:len(pre)-1]
	}
	if len(pre) > 0 {
		pre = append(pre, "")
	}
	out := append(pre, newSec...)
	if insertAt < len(lines) {
		out = append(out, "")
		out = append(out, lines[insertAt:]...)
	}
	return strings.Join(out, "\n")
}
