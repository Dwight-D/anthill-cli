// Package bootstrap carries the install-time / integrity half of the anthill
// binary: the embedded pinned framework template and the logic that scaffolds,
// verifies, and syncs a consumer install against it. The runtime schema-owner
// half (backlog/escalation) lives in internal/backlog and internal/escalation.
package bootstrap

import (
	"embed"
	"io/fs"
)

// TemplateRef is the upstream framework-home commit the embedded template
// payload was vendored from. Stamped into a fresh install's
// .anthill/framework.md synced-through by `scaffold`, and compared by `sync`
// and `doctor`. CI re-vendors template/ and updates this on each tagged CLI
// release.
const TemplateRef = "3797138ddd5eb6e89d083aa001156d4d28fefe18"

// TemplateRepo is the upstream framework home the payload is vendored from.
const TemplateRepo = "https://github.com/Dwight-D/anthill"

// BootstrapDocURL is the canonical, fetchable BOOTSTRAP.md entrypoint that
// `bootstrap` prints. The doc is authoritative and fetched fresh, never
// embedded, so install instructions cannot drift from this binary.
const BootstrapDocURL = "https://raw.githubusercontent.com/Dwight-D/anthill/main/BOOTSTRAP.md"

// templateFS embeds the pinned mechanical-scaffold payload: the ten
// general-tier skills (pristine), the .anthill/ placeholder tree, tools/,
// CLAUDE.template.md, and .gitignore. The `all:` prefix is required so the
// payload's dotfiles and dot-directories (.claude, .anthill, .gitignore) are
// included — a bare //go:embed silently excludes names beginning with '.'.
//
//go:embed all:template
var templateFS embed.FS

// TemplateFS returns the embedded template rooted at the payload top (so paths
// read as ".claude/skills/...", ".anthill/...", "CLAUDE.template.md", etc.).
func TemplateFS() fs.FS {
	sub, err := fs.Sub(templateFS, "template")
	if err != nil {
		// Unreachable: "template" is a compile-time embedded directory.
		panic(err)
	}
	return sub
}
