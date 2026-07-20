package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
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
			ws := "---\nsweep-order: " + strings.Join(streams, ", ") + "\nnever-implicit:\n---\n\n# Backlog workstreams\n"
			files := map[string]string{
				filepath.Join(anthill, "backlog", "CHANGELOG.md"):   "# Improvement Changelog\n",
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

type checkResult struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func (a *App) newDoctorCommand() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Environment + integrity health check (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var checks []checkResult
			root, rerr := a.resolveRoot()
			if rerr != nil {
				checks = append(checks, checkResult{"discoverable", false, rerr.Error()})
				return a.reportDoctor(checks)
			}
			checks = append(checks, checkResult{"discoverable", true, root})

			store := backlog.NewStore(root)
			// Config files present.
			cfg := filepath.Join(root, ".anthill", "backlog", "workstreams.md")
			if _, err := os.Stat(cfg); err == nil {
				checks = append(checks, checkResult{"config-present", true, "workstreams.md found"})
			} else {
				checks = append(checks, checkResult{"config-present", false, "missing workstreams.md"})
			}
			// Sweep-order names existing dirs.
			listed, err := store.ListedSweepOrder()
			if err != nil {
				checks = append(checks, checkResult{"sweep-order", false, err.Error()})
			} else {
				missing := []string{}
				for _, w := range listed {
					if ok, _ := store.IsWorkstream(w); !ok {
						missing = append(missing, w)
					}
				}
				if len(missing) == 0 {
					checks = append(checks, checkResult{"sweep-order", true, "all sweep-order streams have dirs"})
				} else {
					checks = append(checks, checkResult{"sweep-order", false, "no dir for: " + strings.Join(missing, ", ")})
				}
			}
			// backlog validate --strict.
			res, err := store.Validate(true)
			if err != nil {
				checks = append(checks, checkResult{"backlog-valid", false, err.Error()})
			} else if res.OK {
				checks = append(checks, checkResult{"backlog-valid", true, fmt.Sprintf("%d items clean", res.Checked)})
			} else {
				checks = append(checks, checkResult{"backlog-valid", false, fmt.Sprintf("%d violations", len(res.Violations))})
			}
			// Escalations well-formed + no answered-but-unapplied.
			estore, _ := a.escalationStore()
			problems, err := estore.ValidateWellFormed()
			if err != nil {
				checks = append(checks, checkResult{"escalations-valid", false, err.Error()})
			} else if len(problems) == 0 {
				checks = append(checks, checkResult{"escalations-valid", true, "records well-formed"})
			} else {
				checks = append(checks, checkResult{"escalations-valid", false, strings.Join(problems, "; ")})
			}
			unapplied, err := estore.UnappliedAnswered()
			if err != nil {
				checks = append(checks, checkResult{"escalations-applied", false, err.Error()})
			} else if len(unapplied) == 0 {
				checks = append(checks, checkResult{"escalations-applied", true, "no answered-but-unapplied records"})
			} else {
				checks = append(checks, checkResult{"escalations-applied", false, "answered but unapplied: " + strings.Join(unapplied, ", ")})
			}
			_ = strict
			return a.reportDoctor(checks)
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "fail on warnings too")
	return cmd
}

func (a *App) reportDoctor(checks []checkResult) error {
	ok := true
	for _, c := range checks {
		if !c.OK {
			ok = false
		}
	}
	if a.json {
		if err := a.emitJSON(map[string]any{"ok": ok, "checks": checks}); err != nil {
			return err
		}
	} else {
		for _, c := range checks {
			mark := "ok"
			if !c.OK {
				mark = "FAIL"
			}
			a.note("[%s] %s: %s", mark, c.Name, c.Detail)
		}
	}
	if !ok {
		return &Error{Exit: 3, Code: "validation", Message: "doctor found integrity problems"}
	}
	return nil
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
					"escalation_problems": problems,
				}); jerr != nil {
					return jerr
				}
			} else if ok {
				a.note("ok: %d backlog items, escalations well-formed", res.Checked)
			} else {
				for _, v := range res.Violations {
					a.note("%s [%s]: %s", v.ID, v.Check, v.Message)
				}
				for _, p := range problems {
					a.note("escalation %s", p)
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
