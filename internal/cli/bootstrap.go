package cli

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/bootstrap"
	"github.com/Dwight-D/anthill-cli/internal/version"
)

// bootstrapPreamble is the compact, agent-directed instruction printed alongside
// the canonical BOOTSTRAP.md entrypoint.
const bootstrapPreamble = "You are installing Anthill. Read this document, run `anthill scaffold`, " +
	"then drive the derivation session with the user."

type bootstrapView struct {
	Entrypoint string `json:"entrypoint"`
	Preamble   string `json:"preamble"`
}

func (a *App) newBootstrapCommand() *cobra.Command {
	var open bool
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Print the Anthill install entrypoint (or open it in a browser)",
		Long: "bootstrap is the headline entrypoint: it prints the canonical, fetchable " +
			"BOOTSTRAP.md URL plus a short agent-directed preamble. It has zero side effects " +
			"and is safe to run in any directory, in or out of a repository.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			url := bootstrap.BootstrapDocURL
			if open {
				if err := openURL(url); err != nil {
					return internalErr("could not open a browser for " + url + ": " + err.Error())
				}
				a.note("opened %s", url)
				return nil
			}
			if a.json {
				return a.emitJSON(bootstrapView{Entrypoint: url, Preamble: bootstrapPreamble})
			}
			a.answer("%s", url)
			a.answer("")
			a.answer("%s", bootstrapPreamble)
			return nil
		},
	}
	cmd.Flags().BoolVar(&open, "open", false, "open the entrypoint URL in the platform browser instead of printing it")
	return cmd
}

// openURL launches the platform browser on url without waiting.
func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

type scaffoldView struct {
	Written []string `json:"written"`
	Skipped []string `json:"skipped"`
	Refused []string `json:"refused"`
	Ref     string   `json:"ref"`
}

func (a *App) newScaffoldCommand() *cobra.Command {
	var into string
	var force, dryRun bool
	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Write the embedded framework template into a git repo",
		Long: "scaffold performs the mechanical install: it writes the pinned, embedded " +
			"framework template (the general-tier skills, the .anthill/ placeholder tree, " +
			"CLAUDE.template.md, tools/, .gitignore) into the target directory and stamps " +
			".anthill/framework.md with the embedded ref. It is non-destructive: files that " +
			"differ from the template are refused unless --force.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := into
			if target == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return internalErr(err.Error())
				}
				target = cwd
			}
			if !bootstrap.InsideGitRepo(target) {
				return preconditionErr("scaffold target is not inside a git repository: " + target +
					" (run `git init` first)")
			}
			res, err := bootstrap.Scaffold(target, force, dryRun)
			if err != nil {
				return internalErr(err.Error())
			}
			view := scaffoldView{
				Written: orEmpty(res.Written),
				Skipped: orEmpty(res.Skipped),
				Refused: orEmpty(res.Refused),
				Ref:     res.Ref,
			}
			if a.json {
				if jerr := a.emitJSON(view); jerr != nil {
					return jerr
				}
			} else {
				a.printScaffoldManifest(view, dryRun)
			}
			if len(res.Refused) > 0 && !force && !dryRun {
				return validationErr("scaffold refused files that differ from the template; " +
					"re-run with --force to overwrite, or resolve them")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&into, "into", "", "target directory to scaffold into (default: current directory)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite files that differ from the template")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "compute and print the manifest without writing anything")
	return cmd
}

func (a *App) printScaffoldManifest(v scaffoldView, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "(dry-run) "
	}
	a.answer("%sscaffold: %d written, %d skipped (identical), %d refused (differs) — ref %s",
		prefix, len(v.Written), len(v.Skipped), len(v.Refused), shortRef(v.Ref))
	for _, p := range v.Written {
		a.answer("  + %s", p)
	}
	for _, p := range v.Skipped {
		a.answer("  = %s", p)
	}
	for _, p := range v.Refused {
		a.answer("  ! %s (differs — use --force to overwrite)", p)
	}
	if len(v.Refused) == 0 || dryRun {
		a.note("next: now derive .anthill/ with the user — see INSTALLATION.md Steps 3–6")
	}
}

type syncView struct {
	Updated   []string `json:"updated"`
	Unchanged []string `json:"unchanged"`
	Conflicts []string `json:"conflicts"`
	FromRef   string   `json:"from_ref"`
	ToRef     string   `json:"to_ref"`
}

func (a *App) newSyncCommand() *cobra.Command {
	var force, dryRun bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Bring installed general-tier skills up to the embedded template ref",
		Long: "sync diffs the installed general-tier skills against the embedded pinned " +
			"template, re-copies changed skills verbatim, and bumps .anthill/framework.md " +
			"synced-through. The sanctioned autonomous adaptations (proceed-list, decisions-log " +
			"path) are preserved; an upstream change that collides with them is reported as a " +
			"conflict and left unchanged unless --force.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := a.resolveRoot()
			if err != nil {
				return err
			}
			res, serr := bootstrap.Sync(root, dryRun, force)
			if serr != nil {
				return internalErr(serr.Error())
			}
			view := syncView{
				Updated:   orEmpty(res.Updated),
				Unchanged: orEmpty(res.Unchanged),
				Conflicts: orEmpty(res.Conflicts),
				FromRef:   res.FromRef,
				ToRef:     res.ToRef,
			}
			if a.json {
				if jerr := a.emitJSON(view); jerr != nil {
					return jerr
				}
			} else {
				a.printSyncManifest(view, dryRun)
			}
			if len(res.Conflicts) > 0 {
				return validationErr("sync found unresolved conflicts in: " +
					strings.Join(res.Conflicts, ", ") + " (re-run with --force to overwrite)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the skill-level diff without applying it")
	cmd.Flags().BoolVar(&force, "force", false, "apply even when a skill has an unexpected local edit")
	return cmd
}

func (a *App) printSyncManifest(v syncView, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "(dry-run) "
	}
	a.answer("%ssync: %d updated, %d unchanged, %d conflicts — %s → %s",
		prefix, len(v.Updated), len(v.Unchanged), len(v.Conflicts),
		shortRef(v.FromRef), shortRef(v.ToRef))
	for _, s := range v.Updated {
		a.answer("  ~ %s (re-copied verbatim)", s)
	}
	for _, s := range v.Conflicts {
		a.answer("  ! %s (upstream change collides with a local edit — use --force)", s)
	}
}

type versionView struct {
	Version     string `json:"version"`
	TemplateRef string `json:"template_ref"`
}

func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the anthill CLI version and embedded template ref",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.json {
				return a.emitJSON(versionView{
					Version:     version.Version,
					TemplateRef: bootstrap.TemplateRef,
				})
			}
			a.answer("%s", version.String())
			a.answer("template ref: %s", bootstrap.TemplateRef)
			return nil
		},
	}
}

// orEmpty returns a non-nil slice so JSON encodes [] rather than null.
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// shortRef abbreviates a long commit ref for human display; an empty ref renders
// as "manual".
func shortRef(ref string) string {
	if ref == "" {
		return "manual"
	}
	if len(ref) > 12 {
		return ref[:12]
	}
	return ref
}
