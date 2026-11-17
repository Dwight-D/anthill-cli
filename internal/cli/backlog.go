package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
)

// newBacklogCommand builds the `anthill backlog` group.
func (a *App) newBacklogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backlog",
		Short: "Create, list, triage, claim, deliver, and close backlog items",
	}
	cmd.AddCommand(
		a.newBacklogNew(),
		a.newBacklogList(),
		a.newBacklogShow(),
		a.newBacklogSet(),
		a.newBacklogNext(),
		a.newBacklogTriage(),
		a.newBacklogClaim(),
		a.newBacklogDeliver(),
		a.newBacklogBlock(),
		a.newBacklogUnblock(),
		a.newBacklogRelease(),
		a.newBacklogPark(),
		a.newBacklogUnpark(),
		a.newBacklogDispatchVerb("approve", backlog.DispatchApproved, "Approve items for dispatch (user-gated)"),
		a.newBacklogDispatchVerb("fast-track", backlog.DispatchFastTracked, "Approve items and pin them to the top of every queue (user-gated)"),
		a.newBacklogDispatchVerb("unapprove", backlog.DispatchProposed, "Return items to dispatch: proposed"),
		a.newBacklogClose(),
		a.newBacklogValidate(),
		a.newBacklogMigrate(),
		a.newBacklogPrune(),
	)
	return cmd
}

// findItem resolves the store and loads one item.
func (a *App) findItem(id string) (*backlog.Store, *backlog.Item, error) {
	store, err := a.backlogStore()
	if err != nil {
		return nil, nil, err
	}
	it, err := store.Find(id)
	if err != nil {
		return nil, nil, wrapStoreErr(err)
	}
	return store, it, nil
}

// emitTransition prints the result of a single-item transition: the item view
// under --json, else a stderr note and the id on stdout.
func (a *App) emitTransition(it *backlog.Item, format string, args ...any) error {
	if a.json {
		return a.emitJSON(viewItem(it, false))
	}
	a.note(format, args...)
	a.answer("%s", it.ID)
	return nil
}

func checkEnumFlag(key, v string) error {
	if v != "" && !backlog.EnumValues(key)[v] {
		return usageErr("--" + key + " must be one of " + enumList(key))
	}
	return nil
}

func enumList(key string) string {
	var vals []string
	for v := range backlog.EnumValues(key) {
		vals = append(vals, v)
	}
	sortStrings(vals)
	return strings.Join(vals, "|")
}

func (a *App) newBacklogNew() *cobra.Command {
	var title, value, summary, description, descFile, source, hint, priority, reporter string
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
			if err := checkEnumFlag("priority", priority); err != nil {
				return err
			}
			if err := checkEnumFlag("reporter", reporter); err != nil {
				return err
			}
			if description != "" && descFile != "" {
				return usageErr("--description and --description-file are mutually exclusive")
			}
			if descFile != "" {
				data, err := os.ReadFile(descFile)
				if err != nil {
					return usageErr("read --description-file: " + err.Error())
				}
				description = string(data)
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := store.New(backlog.NewParams{
				Title: title, Value: value, Summary: summary, Description: description,
				Source: source, Hint: hint, Priority: priority, Reporter: reporter,
			})
			if err != nil {
				return wrapStoreErr(err)
			}
			return a.emitTransition(it, "created %s", it.ID)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "item title (required)")
	cmd.Flags().StringVar(&value, "value", "", "the pain removed / potential unlocked (required)")
	cmd.Flags().StringVar(&summary, "summary", "", "one or two sentences: what the change is")
	cmd.Flags().StringVar(&description, "description", "", "full description, written under ## Description")
	cmd.Flags().StringVar(&descFile, "description-file", "", "read the description from a file")
	cmd.Flags().StringVar(&source, "source", "", "where it came up")
	cmd.Flags().StringVar(&hint, "hint", "", "non-binding submitter workstream hint")
	cmd.Flags().String("backlog", "", "hidden alias for --hint")
	cmd.Flags().StringVar(&priority, "priority", "", "high|normal|low (default normal)")
	cmd.Flags().StringVar(&reporter, "reporter", "", "user|supervisor|worker (default: $ANTHILL_REPORTER, else inferred)")
	_ = cmd.Flags().MarkHidden("backlog")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("value")
	return cmd
}

func (a *App) newBacklogShow() *cobra.Command {
	var body, noBody bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Print one item's frontmatter and body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			includeBody := body && !noBody
			_, it, err := a.findItem(args[0])
			if err != nil {
				return err
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

// parsePairs splits key=value arguments.
func parsePairs(pairs []string) ([][2]string, error) {
	var out [][2]string
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, usageErr("malformed pair " + p + " (want key=value)")
		}
		out = append(out, [2]string{k, v})
	}
	return out, nil
}

func (a *App) newBacklogSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <id> key=value...",
		Short: "Mutate free frontmatter keys (workstream= moves the file)",
		Long: "Free keys: title summary value source hint note priority disposition change-type risk verify workstream.\n" +
			"status and dispatch move only through their verbs; stamps and claim fields are CLI-owned.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if len(args) == 1 {
				return usageErr("no key=value pairs given")
			}
			pairs, err := parsePairs(args[1:])
			if err != nil {
				return err
			}
			store, it, err := a.findItem(id)
			if err != nil {
				return err
			}
			var moveTo string
			var applied []string
			for _, p := range pairs {
				k, v := p[0], p[1]
				switch {
				case k == "id":
					return validationErr("id is immutable and cannot be set")
				case k == "status":
					return validationErr("status moves only through the lifecycle verbs (triage, claim, deliver, block, unblock, release, park, unpark, close)")
				case k == "dispatch":
					return validationErr("dispatch moves only through approve, fast-track, unapprove")
				case !backlog.IsSettableKey(k):
					return validationErr("unknown or CLI-owned key " + k)
				case k == "workstream":
					if strings.TrimSpace(v) == "" {
						return validationErr("workstream target must be non-empty")
					}
					moveTo = v
				default:
					it.SetField(k, v)
					applied = append(applied, k+"="+v)
				}
			}
			moved := false
			if moveTo != "" && moveTo != it.Workstream {
				if err := store.Reassign(it, moveTo, ""); err != nil {
					return wrapStoreErr(err)
				}
				moved = true
			} else if err := store.Save(it); err != nil {
				return wrapStoreErr(err)
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
		Short: "Print the highest-ranked ready item (without claiming)",
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

func (a *App) newBacklogTriage() *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:   "triage <id> workstream=<ws> summary=… change-type=… risk=… verify=… disposition=… [priority=…]",
		Short: "Triage a new item: set the triaged field set and move it into its workstream",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pairs, err := parsePairs(args[1:])
			if err != nil {
				return err
			}
			fields := map[string]string{}
			for _, p := range pairs {
				fields[p[0]] = p[1]
			}
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			if err := store.Triage(it, fields, note, ""); err != nil {
				return wrapStoreErr(err)
			}
			return a.emitTransition(it, "triaged %s → %s/", it.ID, it.Workstream)
		},
	}
	cmd.Flags().StringVar(&note, "triage-note", "", "triage reasoning, appended under ## Triage")
	return cmd
}

func (a *App) newBacklogClaim() *cobra.Command {
	var next, force bool
	var workstream, by, session string
	cmd := &cobra.Command{
		Use:   "claim <id> | --next",
		Short: "Atomically take ownership of a ready item (status: in-progress)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if next && len(args) > 0 {
				return usageErr("--next is mutually exclusive with a positional id")
			}
			if !next && len(args) == 0 {
				return usageErr("provide an <id> or --next")
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
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
			} else if it, err = store.Find(args[0]); err != nil {
				return wrapStoreErr(err)
			}
			claimed, err := store.Claim(it, by, session, force)
			if err != nil {
				return wrapStoreErr(err)
			}
			a.ledgerClaim(store.Root(), claimed)
			if a.json {
				return a.emitJSON(viewItem(claimed, true))
			}
			a.note("claimed %s by %s", claimed.ID, claimed.ClaimedBy)
			a.answer("%s", claimed.ID)
			if claimed.Body != "" {
				a.answer("%s", claimed.Body)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&next, "next", false, "claim the item next would select")
	cmd.Flags().StringVar(&workstream, "workstream", "", "scope for --next")
	cmd.Flags().StringVar(&by, "by", "", "claimant handle (default: $ANTHILL_ACTOR, else role:session, else OS user)")
	cmd.Flags().StringVar(&session, "session", "", "harness session id (default: $CLAUDE_CODE_SESSION_ID)")
	cmd.Flags().BoolVar(&force, "force", false, "claim a non-ready item or reclaim an orphan")
	return cmd
}

func (a *App) newBacklogDeliver() *cobra.Command {
	var evidence, evidenceFile, by string
	var force bool
	cmd := &cobra.Command{
		Use:   "deliver <id> --evidence \"…\"",
		Short: "Hand a claimed item to validation with its evidence (claimant only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.TrimSpace(evidence)
			if evidenceFile != "" {
				data, err := os.ReadFile(evidenceFile)
				if err != nil {
					return usageErr("read --evidence-file: " + err.Error())
				}
				if text != "" {
					text += "\n"
				}
				text += strings.TrimSpace(string(data))
			}
			if text == "" {
				return usageErr("deliver requires --evidence or --evidence-file")
			}
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			if err := store.Deliver(it, by, text, force); err != nil {
				return wrapStoreErr(err)
			}
			a.ledgerTouch(store.Root(), it.ID, it.Session, "deliver")
			return a.emitTransition(it, "delivered %s (validation)", it.ID)
		},
	}
	cmd.Flags().StringVar(&evidence, "evidence", "", "what was run, exit codes, artifacts")
	cmd.Flags().StringVar(&evidenceFile, "evidence-file", "", "read evidence from a file")
	cmd.Flags().StringVar(&by, "by", "", "caller handle (default: $ANTHILL_ACTOR, else role:session, else OS user)")
	cmd.Flags().BoolVar(&force, "force", false, "deliver on behalf of another claimant")
	return cmd
}

func (a *App) newBacklogBlock() *cobra.Command {
	return &cobra.Command{
		Use:   "block <id> \"<reason>\"",
		Short: "Mark an in-progress item blocked (reason logged)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[1]) == "" {
				return usageErr("block requires a reason")
			}
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			if err := store.Block(it, args[1], ""); err != nil {
				return wrapStoreErr(err)
			}
			a.ledgerTouch(store.Root(), it.ID, it.Session, "block")
			return a.emitTransition(it, "blocked %s: %s", it.ID, args[1])
		},
	}
}

func (a *App) newBacklogUnblock() *cobra.Command {
	var release bool
	cmd := &cobra.Command{
		Use:   "unblock <id>",
		Short: "Resume a blocked item under the same claim (--release returns it to triaged)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			if err := store.Unblock(it, release, ""); err != nil {
				return wrapStoreErr(err)
			}
			return a.emitTransition(it, "unblocked %s (%s)", it.ID, it.Status)
		},
	}
	cmd.Flags().BoolVar(&release, "release", false, "drop the claim and return the item to triaged")
	return cmd
}

func (a *App) newBacklogRelease() *cobra.Command {
	return &cobra.Command{
		Use:   "release <id>",
		Short: "Drop the claim on an in-progress item (back to triaged)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			sid := it.Session
			if err := store.Release(it, ""); err != nil {
				return wrapStoreErr(err)
			}
			a.ledgerTouch(store.Root(), it.ID, sid, "release")
			return a.emitTransition(it, "released %s", it.ID)
		},
	}
}

func (a *App) newBacklogPark() *cobra.Command {
	return &cobra.Command{
		Use:   "park <id> \"<reason>\"",
		Short: "Defer a new or triaged item (status: parked)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[1]) == "" {
				return usageErr("park requires a reason")
			}
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			if err := store.Park(it, args[1], ""); err != nil {
				return wrapStoreErr(err)
			}
			return a.emitTransition(it, "parked %s", it.ID)
		},
	}
}

func (a *App) newBacklogUnpark() *cobra.Command {
	return &cobra.Command{
		Use:   "unpark <id>",
		Short: "Return a parked item to its prior status (new in intake, else triaged)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			if err := store.Unpark(it, ""); err != nil {
				return wrapStoreErr(err)
			}
			return a.emitTransition(it, "unparked %s (%s)", it.ID, it.Status)
		},
	}
}

// newBacklogDispatchVerb builds approve / fast-track / unapprove: each sets
// dispatch on one or more ids. Every id is resolved before any is written.
func (a *App) newBacklogDispatchVerb(use, value, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use + " <id>...",
		Short: short,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			var items []*backlog.Item
			for _, id := range args {
				it, err := store.Find(id)
				if err != nil {
					return wrapStoreErr(err)
				}
				if it.Terminal() {
					return preconditionErr("item " + id + " is " + it.Status + " (archived); dispatch cannot change")
				}
				items = append(items, it)
			}
			for _, it := range items {
				changed, err := store.SetDispatch(it, value, "")
				if err != nil {
					return wrapStoreErr(err)
				}
				if changed {
					a.note("%s: dispatch %s", it.ID, value)
				} else {
					a.note("%s: already %s", it.ID, value)
				}
			}
			if a.json {
				return a.emitJSON(viewItems(items))
			}
			for _, it := range items {
				a.answer("%s", it.ID)
			}
			return nil
		},
	}
	// --yes is accepted and ignored: the verb itself is the gate.
	cmd.Flags().Bool("yes", false, "accepted for compatibility; the verb is the gate")
	_ = cmd.Flags().MarkHidden("yes")
	return cmd
}

func (a *App) newBacklogClose() *cobra.Command {
	// The mode flags are pure booleans: they never consume a value, so the id
	// parses as the positional in either order, and the reason is the second
	// positional (`close <id> --reject "reason"`).
	var done, reject, discard, remove, force bool
	cmd := &cobra.Command{
		Use:   "close <id> [\"reason\"]",
		Short: "Close an item: --done (archive) | --reject (back to in-progress) | --discard (archive) | --remove (delete)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("block") {
				return usageErr("close --block is gone: use 'anthill backlog block <id> \"<reason>\"'")
			}
			id := args[0]
			reason := ""
			if len(args) == 2 {
				reason = args[1]
			}
			var mode string
			n := 0
			for m, set := range map[string]bool{
				backlog.CloseDone: done, backlog.CloseReject: reject,
				backlog.CloseDiscard: discard, backlog.CloseRemove: remove,
			} {
				if set {
					mode = m
					n++
				}
			}
			if n == 0 {
				return usageErr("exactly one mode required: --done | --reject | --discard | --remove")
			}
			if n > 1 {
				return usageErr("close mode flags are mutually exclusive")
			}
			if mode == backlog.CloseReject && strings.TrimSpace(reason) == "" {
				return usageErr("close --reject requires a reason")
			}
			if force && mode != backlog.CloseDone {
				return usageErr("--force applies only to close --done")
			}
			store, it, err := a.findItem(id)
			if err != nil {
				return err
			}
			sid := it.Session
			changelog, err := store.Close(it, mode, reason, "", force)
			if err != nil {
				return wrapStoreErr(err)
			}
			a.ledgerTouch(store.Root(), id, sid, "close")
			if a.json {
				return a.emitJSON(map[string]any{
					"id": id, "mode": mode, "status": it.Status, "changelog": changelog,
				})
			}
			a.note("closed %s (%s)", id, mode)
			return nil
		},
	}
	cmd.Flags().BoolVar(&done, "done", false, "validated: archive in done/")
	cmd.Flags().BoolVar(&reject, "reject", false, "evidence insufficient: back to in-progress (reason required)")
	cmd.Flags().BoolVar(&discard, "discard", false, "not worth doing: archive in discarded/")
	cmd.Flags().BoolVar(&remove, "remove", false, "filed in error: delete the file")
	cmd.Flags().BoolVar(&force, "force", false, "close --done from a status other than validation")
	cmd.Flags().Bool("block", false, "removed; use 'backlog block'")
	_ = cmd.Flags().MarkHidden("block")
	return cmd
}

func (a *App) newBacklogValidate() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Certify the backlog tree against the schema; lists every violation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runBacklogValidate(strict)
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "promote warnings (change-type vocabulary) to violations")
	return cmd
}

func (a *App) runBacklogValidate(strict bool) error {
	store, err := a.backlogStore()
	if err != nil {
		return err
	}
	res, err := store.Validate(strict)
	if err != nil {
		return wrapStoreErr(err)
	}
	if a.json {
		if jerr := a.emitJSON(res); jerr != nil {
			return jerr
		}
	} else {
		a.printViolations(res)
	}
	if !res.OK {
		return validationErr("backlog validation found " + itoa(len(res.Violations)) + " violation(s)")
	}
	return nil
}

// printViolations prints every violation as "<id>: <rule>: <message>" on
// stdout, then warnings, then a one-line summary on stderr.
func (a *App) printViolations(res *backlog.ValidateResult) {
	for _, v := range res.Violations {
		a.answer("%s", v.String())
	}
	for _, w := range res.Warnings {
		a.answer("warn: %s", w.String())
	}
	if res.OK {
		a.note("ok: %d items", res.Checked)
	}
}

// selectNext returns the highest-ranked ready item, or nil. Intake items never
// qualify; never-implicit workstreams qualify only when named.
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
		if !it.Ready() || it.Workstream == "" {
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
	backlog.SortItems(ready, "rank", order, false)
	return ready[0], nil
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
