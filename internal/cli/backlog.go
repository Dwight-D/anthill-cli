package cli

import (
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
)

// newBacklogCommand builds the `anthill backlog` group.
func (a *App) newBacklogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backlog",
		Short: "Create, list, triage, claim, and close backlog items",
	}
	cmd.AddCommand(
		a.newBacklogNew(),
		a.newBacklogList(),
		a.newBacklogShow(),
		a.newBacklogSet(),
		a.newBacklogNext(),
		a.newBacklogClaim(),
		a.newBacklogClose(),
		a.newBacklogApprove(),
		a.newBacklogValidate(),
	)
	return cmd
}

func (a *App) newBacklogNew() *cobra.Command {
	var title, value, source, hint, priority string
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Submit an intake item (title + value are the whole required ask)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --backlog is a hidden alias for --hint.
			if hint == "" {
				if b, _ := cmd.Flags().GetString("backlog"); b != "" {
					hint = b
				}
			}
			if priority != "" && priority != "high" && priority != "normal" {
				return usageErr("--priority must be high or normal")
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := store.New(backlog.NewParams{
				Title: title, Value: value, Source: source, Hint: hint, Priority: priority,
			})
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(viewItem(it, false))
			}
			a.note("created %s", it.ID)
			a.answer("%s", it.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "item title (required)")
	cmd.Flags().StringVar(&value, "value", "", "the pain removed / potential unlocked (required)")
	cmd.Flags().StringVar(&source, "source", "", "where it came up")
	cmd.Flags().StringVar(&hint, "hint", "", "non-binding submitter workstream hint")
	cmd.Flags().String("backlog", "", "hidden alias for --hint")
	cmd.Flags().StringVar(&priority, "priority", "", "high|normal (triage normally sets this)")
	_ = cmd.Flags().MarkHidden("backlog")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("value")
	return cmd
}

func (a *App) newBacklogList() *cobra.Command {
	var workstream string
	var untriaged, ready bool
	var status []string
	var sortMode string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List items across intake and workstreams",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			if workstream != "" {
				ok, werr := store.IsWorkstream(workstream)
				if werr != nil {
					return internalErr(werr.Error())
				}
				if !ok {
					return notFoundErr("no workstream "+workstream, workstream)
				}
			}
			items, err := store.LoadAll()
			if err != nil {
				return internalErr(err.Error())
			}
			statusSet := toSet(status)
			var filtered []*backlog.Item
			for _, it := range items {
				if workstream != "" && it.Workstream != workstream {
					continue
				}
				if untriaged && it.Workstream != "" {
					continue
				}
				if ready && !it.Ready() {
					continue
				}
				if len(statusSet) > 0 && !statusSet[it.Status] {
					continue
				}
				filtered = append(filtered, it)
			}
			order, _, err := store.SweepOrder()
			if err != nil {
				return internalErr(err.Error())
			}
			sortItems(filtered, sortMode, order)
			if a.json {
				return a.emitJSON(viewItems(filtered))
			}
			if len(filtered) == 0 {
				a.note("no items match")
				return nil
			}
			a.printItemTable(filtered)
			return nil
		},
	}
	cmd.Flags().StringVar(&workstream, "workstream", "", "restrict to one workstream")
	cmd.Flags().BoolVar(&untriaged, "untriaged", false, "only intake items (no workstream)")
	cmd.Flags().BoolVar(&ready, "ready", false, "only dispatchable items")
	cmd.Flags().StringSliceVar(&status, "status", nil, "filter by lifecycle status (repeatable)")
	cmd.Flags().StringVar(&sortMode, "sort", "sweep", "sweep|priority|id")
	return cmd
}

func (a *App) newBacklogShow() *cobra.Command {
	var body, noBody bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Print one item's frontmatter and body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			includeBody := body
			if noBody {
				includeBody = false
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := store.Find(args[0])
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(viewItem(it, includeBody))
			}
			a.printItemDetail(it, includeBody)
			return nil
		},
	}
	cmd.Flags().BoolVar(&body, "body", true, "include the markdown body")
	cmd.Flags().BoolVar(&noBody, "no-body", false, "omit the markdown body")
	return cmd
}

func (a *App) newBacklogSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <id> key=value...",
		Short: "Mutate frontmatter keys (workstream= moves the file)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			pairs := args[1:]
			if len(pairs) == 0 {
				return usageErr("no key=value pairs given")
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := store.Find(id)
			if err != nil {
				return wrapStoreErr(err)
			}
			origWs := it.Workstream
			var moveTo string
			haveMove := false
			var applied []string
			for _, p := range pairs {
				k, v, ok := strings.Cut(p, "=")
				if !ok {
					return usageErr("malformed pair " + p + " (want key=value)")
				}
				if k == "id" {
					return validationErr("id is immutable and cannot be set")
				}
				if !settableKeyOK(k) {
					return validationErr("unknown or immutable key " + k)
				}
				if k == "status" && v == "approved" {
					return validationErr("status=approved must go through 'anthill backlog approve <id> --yes'")
				}
				if k == "workstream" {
					if strings.TrimSpace(v) == "" {
						return validationErr("workstream target must be non-empty")
					}
					moveTo = v
					haveMove = true
					continue
				}
				applyItemKey(it, k, v)
				applied = append(applied, k+"="+v)
			}
			moved := false
			if haveMove && moveTo != origWs {
				if origWs == "" {
					it.Hint = "" // hint is stripped on triage out of intake
				}
				if err := store.Move(it, moveTo); err != nil {
					return wrapStoreErr(err)
				}
				moved = true
			} else {
				if err := store.Save(it); err != nil {
					return wrapStoreErr(err)
				}
			}
			if a.json {
				return a.emitJSON(viewItem(it, false))
			}
			if len(applied) > 0 {
				a.note("set %s: %s", id, strings.Join(applied, " "))
			}
			if moved {
				a.note("moved %s → %s/", id, moveTo)
			}
			a.answer("%s", id)
			return nil
		},
	}
	return cmd
}

func (a *App) newBacklogNext() *cobra.Command {
	var workstream string
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Print the next dispatchable item in sweep order (without claiming)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := a.selectNext(store, workstream)
			if err != nil {
				return err
			}
			if it == nil {
				if a.json {
					return a.emitJSON(nil)
				}
				a.note("no ready items")
				return nil
			}
			if a.json {
				return a.emitJSON(viewItem(it, false))
			}
			a.answer("%s\t%s", it.ID, it.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&workstream, "workstream", "", "restrict to one workstream")
	return cmd
}

func (a *App) newBacklogClaim() *cobra.Command {
	var next bool
	var workstream string
	var force bool
	cmd := &cobra.Command{
		Use:   "claim <id> | --next",
		Short: "Atomically take ownership of an item (status: in-progress)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			if next && len(args) > 0 {
				return usageErr("--next is mutually exclusive with a positional id")
			}
			if !next && len(args) == 0 {
				return usageErr("provide an <id> or --next")
			}
			var it *backlog.Item
			if next {
				it, err = a.selectNext(store, workstream)
				if err != nil {
					return err
				}
				if it == nil {
					return notFoundErr("no ready items to claim", "")
				}
			} else {
				it, err = store.Find(args[0])
				if err != nil {
					return wrapStoreErr(err)
				}
			}
			if it.Status == "in-progress" {
				if !force {
					return conflictErr("item " + it.ID + " is already claimed (in-progress); use --force to reclaim an orphan")
				}
			} else if !it.Ready() && !force {
				return preconditionErr("item " + it.ID + " is not ready (needs approved + verify); use --force to override")
			}
			// Compare-and-set: re-read and confirm the status is unchanged.
			expected := it.Status
			fresh, err := store.Find(it.ID)
			if err != nil {
				return wrapStoreErr(err)
			}
			if fresh.Status != expected {
				return conflictErr("item " + it.ID + " changed under us (expected " + expected + ", found " + fresh.Status + ")")
			}
			fresh.Status = "in-progress"
			fresh.ClaimedAt = time.Now().UTC().Format(time.RFC3339)
			if err := store.Save(fresh); err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(viewItem(fresh, true))
			}
			a.note("claimed %s", fresh.ID)
			a.answer("%s", fresh.ID)
			if fresh.Body != "" {
				a.answer("%s", fresh.Body)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&next, "next", false, "claim the item next would select")
	cmd.Flags().StringVar(&workstream, "workstream", "", "scope for --next")
	cmd.Flags().BoolVar(&force, "force", false, "claim a non-ready item or reclaim an orphan")
	return cmd
}

func (a *App) newBacklogClose() *cobra.Command {
	var done bool
	var discard, remove, block string
	cmd := &cobra.Command{
		Use:   "close <id>",
		Short: "Terminate (--done/--discard/--remove) or park (--block) a claimed item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			n := 0
			if done {
				n++
			}
			if discard != "" {
				n++
			}
			if remove != "" {
				n++
			}
			if block != "" {
				n++
			}
			if n == 0 {
				return usageErr("exactly one disposition required: --done | --discard | --remove | --block")
			}
			if n > 1 {
				return usageErr("disposition flags are mutually exclusive")
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := store.Find(id)
			if err != nil {
				return wrapStoreErr(err)
			}
			var disposition string
			changelog := false
			switch {
			case done:
				if it.Status != "in-progress" {
					return preconditionErr("item " + id + " is " + it.Status + ", not in-progress; only a claimed item can be closed --done")
				}
				if err := store.Delete(it); err != nil {
					return internalErr(err.Error())
				}
				if err := store.AppendChangelog(id, "done", it.Title); err != nil {
					return internalErr(err.Error())
				}
				disposition, changelog = "done", true
			case discard != "":
				if err := store.Delete(it); err != nil {
					return internalErr(err.Error())
				}
				if err := store.AppendChangelog(id, "discarded", discard); err != nil {
					return internalErr(err.Error())
				}
				disposition, changelog = "discard", true
			case remove != "":
				if err := store.Delete(it); err != nil {
					return internalErr(err.Error())
				}
				if err := store.AppendChangelog(id, "removed", remove); err != nil {
					return internalErr(err.Error())
				}
				disposition, changelog = "remove", true
			case block != "":
				if it.Status != "in-progress" {
					return preconditionErr("item " + id + " is " + it.Status + ", not in-progress; only a claimed item can be blocked")
				}
				it.Status = "blocked"
				it.Note = block
				if err := store.Save(it); err != nil {
					return wrapStoreErr(err)
				}
				disposition, changelog = "block", false
			}
			if a.json {
				return a.emitJSON(map[string]any{
					"id": id, "disposition": disposition, "changelog": changelog,
				})
			}
			if disposition == "block" {
				a.note("blocked %s: %s", id, block)
			} else {
				a.note("closed %s (%s)", id, disposition)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&done, "done", false, "completed successfully (terminal)")
	cmd.Flags().StringVar(&discard, "discard", "", "decided not worth doing (terminal)")
	cmd.Flags().StringVar(&remove, "remove", "", "obsolete/superseded/rejected (terminal)")
	cmd.Flags().StringVar(&block, "block", "", "cannot proceed now (non-terminal, status: blocked)")
	return cmd
}

func (a *App) newBacklogApprove() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve an item for dispatch (gated: requires --yes human confirmation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !yes {
				return preconditionErr("approval is a gated human action; re-run with --yes to confirm")
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := store.Find(id)
			if err != nil {
				return wrapStoreErr(err)
			}
			if it.Status != "idea" {
				return preconditionErr("item " + id + " is " + it.Status + "; only an item in status idea can be approved")
			}
			it.Status = "approved"
			if err := store.Save(it); err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(viewItem(it, false))
			}
			a.note("approved %s", id)
			a.answer("%s", id)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm this human-gated approval")
	return cmd
}

func (a *App) newBacklogValidate() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Certify the backlog tree as schema-well-formed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runBacklogValidate(strict)
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "add cross-field consistency checks")
	return cmd
}

func (a *App) runBacklogValidate(strict bool) error {
	store, err := a.backlogStore()
	if err != nil {
		return err
	}
	res, err := store.Validate(strict)
	if err != nil {
		return internalErr(err.Error())
	}
	if a.json {
		if jerr := a.emitJSON(res); jerr != nil {
			return jerr
		}
	} else if res.OK {
		a.note("ok: %d items", res.Checked)
	} else {
		for _, v := range res.Violations {
			a.note("%s [%s]: %s", v.ID, v.Check, v.Message)
		}
	}
	if !res.OK {
		return validationErr("backlog validation found violations")
	}
	return nil
}

// selectNext returns the next dispatchable (ready) item in sweep order, or nil.
func (a *App) selectNext(store *backlog.Store, workstream string) (*backlog.Item, error) {
	if workstream != "" {
		ok, err := store.IsWorkstream(workstream)
		if err != nil {
			return nil, internalErr(err.Error())
		}
		if !ok {
			return nil, notFoundErr("no workstream "+workstream, workstream)
		}
	}
	items, err := store.LoadAll()
	if err != nil {
		return nil, internalErr(err.Error())
	}
	order, neverImplicit, err := store.SweepOrder()
	if err != nil {
		return nil, internalErr(err.Error())
	}
	var ready []*backlog.Item
	for _, it := range items {
		if !it.Ready() {
			continue
		}
		if it.Workstream == "" {
			continue
		}
		if workstream != "" {
			if it.Workstream != workstream {
				continue
			}
		} else if neverImplicit[it.Workstream] {
			continue
		}
		ready = append(ready, it)
	}
	if len(ready) == 0 {
		return nil, nil
	}
	sortItems(ready, "sweep", order)
	return ready[0], nil
}

func sortItems(items []*backlog.Item, mode string, order []string) {
	wsIndex := map[string]int{}
	for i, w := range order {
		wsIndex[w] = i
	}
	rank := func(it *backlog.Item) int {
		if it.Workstream == "" {
			return len(order) + 1
		}
		if i, ok := wsIndex[it.Workstream]; ok {
			return i
		}
		return len(order)
	}
	prio := func(it *backlog.Item) int {
		if it.Priority == "high" {
			return 0
		}
		return 1
	}
	sort.SliceStable(items, func(i, j int) bool {
		switch mode {
		case "id":
			return items[i].ID < items[j].ID
		case "priority":
			if prio(items[i]) != prio(items[j]) {
				return prio(items[i]) < prio(items[j])
			}
			return items[i].ID < items[j].ID
		default: // sweep
			if rank(items[i]) != rank(items[j]) {
				return rank(items[i]) < rank(items[j])
			}
			if prio(items[i]) != prio(items[j]) {
				return prio(items[i]) < prio(items[j])
			}
			return items[i].ID < items[j].ID
		}
	})
}

func toSet(vals []string) map[string]bool {
	if len(vals) == 0 {
		return nil
	}
	m := map[string]bool{}
	for _, v := range vals {
		m[v] = true
	}
	return m
}
