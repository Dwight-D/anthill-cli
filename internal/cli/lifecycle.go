package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/bootstrap"
)

// defaultWorkstreams are the streams init seeds by default.
var defaultWorkstreams = []string{"bugs", "cli", "dev", "process"}

func (a *App) newInitCommand() *cobra.Command {
	var extra []string
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a fresh .anthill/ tree",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := a.rootFlag
			if target == "" {
				if env := os.Getenv("ANTHILL_ROOT"); env != "" {
					target = env
				} else {
					cwd, err := os.Getwd()
					if err != nil {
						return internalErr(err.Error())
					}
					target = cwd
				}
			}
			anthill := filepath.Join(target, ".anthill")
			if _, err := os.Stat(anthill); err == nil && !force {
				return preconditionErr(".anthill/ already exists at " + target + "; pass --force to populate without clobbering")
			}
			streams := append([]string{}, defaultWorkstreams...)
			for _, w := range extra {
				if w != "" && !contains(streams, w) {
					streams = append(streams, w)
				}
			}
			dirs := []string{
				filepath.Join(anthill, "backlog", "intake"),
				filepath.Join(anthill, "escalations"),
			}
			for _, w := range streams {
				dirs = append(dirs, filepath.Join(anthill, "backlog", w))
			}
			for _, d := range dirs {
				if err := os.MkdirAll(d, 0o755); err != nil {
					return internalErr(err.Error())
				}
			}
			ws := "---\n" +
				"sweep-order: " + strings.Join(streams, ", ") + "\n" +
				"never-implicit:\n" +
				"change-types: doc, tooling, bugfix, audit, verify, rename, removal, design, subjective\n" +
				"never-auto-change-types: rename, removal, design, subjective\n" +
				"---\n\n# Backlog workstreams\n"
			files := map[string]string{
				filepath.Join(anthill, "backlog", "CHANGELOG.md"):   "# Improvement Changelog\n\nOne line per closed item, newest first.\n\n## Done\n\n## Discarded (triaged out, not done)\n",
				filepath.Join(anthill, "backlog", "workstreams.md"): ws,
				filepath.Join(anthill, "escalations", "LOG.md"):     "# Escalations log\n",
			}
			for path, content := range files {
				if _, err := os.Stat(path); err == nil {
					continue // never overwrite an existing config file
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					return internalErr(err.Error())
				}
			}
			if a.json {
				return a.emitJSON(map[string]any{
					"root": target, "workstreams": streams, "initialized": true,
				})
			}
			a.note("initialized .anthill/ at %s (workstreams: %s)", target, strings.Join(streams, ", "))
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&extra, "workstream", nil, "seed an extra workstream (repeatable)")
	cmd.Flags().BoolVar(&force, "force", false, "populate an existing dir without clobbering present files")
	return cmd
}

// severity classifies how an unhealthy check affects the doctor exit code.
type severity int

const (
	sevHard   severity = iota // unhealthy → fails doctor unconditionally
	sevStrict                 // unhealthy → fails only under --strict
	sevInfo                   // informational; never fails doctor
)

// checkResult is one doctor check. Section is "A" (install integrity) or "B"
// (runtime data integrity). The severity governs exit but is not serialized.
type checkResult struct {
	Section string `json:"section"`
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail"`
	sev     severity
}

func (a *App) newDoctorCommand() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Install-integrity + runtime data health check (read-only)",
		Long: "doctor reports two labeled sections: Section A checks install integrity " +
			"(general-tier skills byte-identical to the pinned template, expected .anthill/ " +
			"structure, remaining derivation placeholders, sync status) and Section B checks " +
			"runtime data integrity (discoverable tree, config present, sweep order, backlog " +
			"and escalation validity). --strict also fails on remaining placeholders.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, rerr := a.resolveRoot()
			if rerr != nil {
				checks := []checkResult{{"B", "discoverable", false, rerr.Error(), sevHard}}
				return a.reportDoctor(checks, strict)
			}
			var checks []checkResult
			checks = append(checks, a.sectionAChecks(root)...)
			checks = append(checks, a.sectionBChecks(root)...)
			return a.reportDoctor(checks, strict)
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "also fail on remaining template placeholders")
	return cmd
}

// sectionAChecks builds the install-integrity checks against the pinned template.
func (a *App) sectionAChecks(root string) []checkResult {
	var checks []checkResult

	// skill integrity — every general-tier skill byte-identical to the pinned
	// template; there are no exempted regions.
	if skills, err := bootstrap.CheckSkillIntegrity(root); err != nil {
		checks = append(checks, checkResult{"A", "skill-integrity", false, err.Error(), sevHard})
	} else {
		var bad []string
		for _, s := range skills {
			if !s.OK {
				bad = append(bad, s.Name+" ("+s.Detail+")")
			}
		}
		if len(bad) == 0 {
			checks = append(checks, checkResult{"A", "skill-integrity", true,
				fmt.Sprintf("%d general-tier skills match the pinned template", len(skills)), sevHard})
		} else {
			checks = append(checks, checkResult{"A", "skill-integrity", false,
				"illegal local edits — " + strings.Join(bad, "; "), sevHard})
		}
	}

	// structure — expected .anthill/ tree present.
	if missing, err := bootstrap.CheckStructure(root); err != nil {
		checks = append(checks, checkResult{"A", "structure", false, err.Error(), sevHard})
	} else if len(missing) == 0 {
		checks = append(checks, checkResult{"A", "structure", true, "expected .anthill/ tree present", sevHard})
	} else {
		checks = append(checks, checkResult{"A", "structure", false, "missing: " + strings.Join(missing, ", "), sevHard})
	}

	// derivation status — remaining template placeholders (info unless --strict).
	if undrived, err := bootstrap.DerivationStatus(root); err != nil {
		checks = append(checks, checkResult{"A", "derivation-status", false, err.Error(), sevHard})
	} else if len(undrived) == 0 {
		checks = append(checks, checkResult{"A", "derivation-status", true, "no remaining template placeholders", sevStrict})
	} else {
		checks = append(checks, checkResult{"A", "derivation-status", false,
			fmt.Sprintf("%d file(s) still hold template placeholders: %s", len(undrived), strings.Join(undrived, ", ")), sevStrict})
	}

	// sync status — installed synced-through vs embedded ref (informational).
	if st, err := bootstrap.SyncStatus(root); err != nil {
		checks = append(checks, checkResult{"A", "sync-status", false, err.Error(), sevInfo})
	} else {
		checks = append(checks, checkResult{"A", "sync-status", st.UpToDate, st.Detail, sevInfo})
	}
	return checks
}

// sectionBChecks builds the runtime data-integrity checks (unchanged semantics
// from the original doctor).
func (a *App) sectionBChecks(root string) []checkResult {
	var checks []checkResult
	checks = append(checks, checkResult{"B", "discoverable", true, root, sevHard})

	store := backlog.NewStore(root)
	cfg := filepath.Join(root, ".anthill", "backlog", "workstreams.md")
	if _, err := os.Stat(cfg); err == nil {
		checks = append(checks, checkResult{"B", "config-present", true, "workstreams.md found", sevHard})
	} else {
		checks = append(checks, checkResult{"B", "config-present", false, "missing workstreams.md", sevHard})
	}

	listed, err := store.ListedSweepOrder()
	if err != nil {
		checks = append(checks, checkResult{"B", "sweep-order", false, err.Error(), sevHard})
	} else {
		missing := []string{}
		for _, w := range listed {
			if ok, _ := store.IsWorkstream(w); !ok {
				missing = append(missing, w)
			}
		}
		if len(missing) == 0 {
			checks = append(checks, checkResult{"B", "sweep-order", true, "all sweep-order streams have dirs", sevHard})
		} else {
			checks = append(checks, checkResult{"B", "sweep-order", false, "no dir for: " + strings.Join(missing, ", "), sevHard})
		}
	}

	res, err := store.Validate(true)
	if err != nil {
		checks = append(checks, checkResult{"B", "backlog-valid", false, err.Error(), sevHard})
	} else {
		if res.OK {
			checks = append(checks, checkResult{"B", "backlog-valid", true, fmt.Sprintf("%d items clean", res.Checked), sevHard})
		} else {
			checks = append(checks, checkResult{"B", "backlog-valid", false, fmt.Sprintf("%d violations", len(res.Violations)), sevHard})
		}
		// change-type vocabulary: advisory only. An item using a change-type
		// outside the project's declared set warns but never fails doctor —
		// the vocabulary is the project's to own (declared in workstreams.md).
		if len(res.Warnings) == 0 {
			checks = append(checks, checkResult{"B", "change-type-vocab", true, "all change-types within declared vocabulary", sevInfo})
		} else {
			var ids []string
			for _, w := range res.Warnings {
				ids = append(ids, w.ID)
			}
			checks = append(checks, checkResult{"B", "change-type-vocab", false,
				fmt.Sprintf("%d item(s) use an undeclared change-type: %s", len(res.Warnings), strings.Join(ids, ", ")), sevInfo})
		}
	}

	estore, _ := a.escalationStore()
	problems, err := estore.ValidateWellFormed()
	if err != nil {
		checks = append(checks, checkResult{"B", "escalations-valid", false, err.Error(), sevHard})
	} else if len(problems) == 0 {
		checks = append(checks, checkResult{"B", "escalations-valid", true, "records well-formed", sevHard})
	} else {
		checks = append(checks, checkResult{"B", "escalations-valid", false, strings.Join(problems, "; "), sevHard})
	}

	unapplied, err := estore.UnappliedAnswered()
	if err != nil {
		checks = append(checks, checkResult{"B", "escalations-applied", false, err.Error(), sevHard})
	} else if len(unapplied) == 0 {
		checks = append(checks, checkResult{"B", "escalations-applied", true, "no answered-but-unapplied records", sevHard})
	} else {
		checks = append(checks, checkResult{"B", "escalations-applied", false, "answered but unapplied: " + strings.Join(unapplied, ", "), sevHard})
	}
	return checks
}

func (a *App) reportDoctor(checks []checkResult, strict bool) error {
	fail := false
	for _, c := range checks {
		if c.OK {
			continue
		}
		switch c.sev {
		case sevHard:
			fail = true
		case sevStrict:
			if strict {
				fail = true
			}
		}
	}
	if a.json {
		if err := a.emitJSON(map[string]any{"ok": !fail, "checks": checks}); err != nil {
			return err
		}
	} else {
		a.printDoctorSections(checks, strict)
	}
	if fail {
		return &Error{Exit: 3, Code: "validation", Message: "doctor found integrity problems"}
	}
	return nil
}

// doctorMark renders a check's status label: "ok" when healthy, "FAIL" when it
// contributes to a non-zero exit, and "warn" when it is unhealthy but only
// informational (never fails, or fails only under --strict which is off here).
func doctorMark(c checkResult, strict bool) string {
	if c.OK {
		return "ok"
	}
	switch c.sev {
	case sevHard:
		return "FAIL"
	case sevStrict:
		if strict {
			return "FAIL"
		}
		return "warn"
	default: // sevInfo
		return "warn"
	}
}

// printDoctorSections renders the checks grouped under their two section headers.
func (a *App) printDoctorSections(checks []checkResult, strict bool) {
	sections := []struct{ key, label string }{
		{"A", "Section A — install integrity"},
		{"B", "Section B — runtime data integrity"},
	}
	for _, sec := range sections {
		printed := false
		for _, c := range checks {
			if c.Section != sec.key {
				continue
			}
			if !printed {
				a.answer("%s", sec.label)
				printed = true
			}
			a.answer("  [%s] %s: %s", doctorMark(c, strict), c.Name, c.Detail)
		}
	}
}

func (a *App) newValidateCommand() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the whole tree (backlog + escalations)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			res, err := store.Validate(strict)
			if err != nil {
				return internalErr(err.Error())
			}
			estore, _ := a.escalationStore()
			problems, eerr := estore.ValidateWellFormed()
			if eerr != nil {
				return internalErr(eerr.Error())
			}
			ok := res.OK && len(problems) == 0
			if a.json {
				if jerr := a.emitJSON(map[string]any{
					"ok":                  ok,
					"checked":             res.Checked,
					"violations":          res.Violations,
					"warnings":            res.Warnings,
					"escalation_problems": problems,
				}); jerr != nil {
					return jerr
				}
			} else {
				if ok {
					a.note("ok: %d backlog items, escalations well-formed", res.Checked)
				} else {
					for _, v := range res.Violations {
						a.note("%s [%s]: %s", v.ID, v.Check, v.Message)
					}
					for _, p := range problems {
						a.note("escalation %s", p)
					}
				}
				// Warnings are advisory — printed regardless of ok, never affect exit.
				for _, w := range res.Warnings {
					a.note("warn: %s [%s]: %s", w.ID, w.Check, w.Message)
				}
			}
			if !ok {
				return validationErr("tree validation found violations")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "add cross-field consistency checks")
	return cmd
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
