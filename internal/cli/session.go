package cli

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/session"
)

// defaultSessionPruneAge is the staleness threshold `session prune` applies
// when --older-than is not given.
const defaultSessionPruneAge = "7d"

// sessionStore resolves the root and returns the sessions ledger store.
func (a *App) sessionStore() (*session.Store, error) {
	root, err := a.resolveRoot()
	if err != nil {
		return nil, err
	}
	return session.NewStore(root), nil
}

// wrapSessionErr translates session package errors into a *Error.
func wrapSessionErr(err error) *Error {
	if err == nil {
		return nil
	}
	var ve *session.ValidationError
	if errors.As(err, &ve) {
		return usageErr(ve.Msg)
	}
	if errors.Is(err, session.ErrNotFound) {
		return notFoundErr(err.Error(), "")
	}
	return internalErr(err.Error())
}

// newSessionCommand builds the `anthill session` group: the runtime-only
// ledger correlating harness sessions with the items they work (§5).
func (a *App) newSessionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Maintain the sessions ledger (which harness session works which item)",
	}
	cmd.AddCommand(
		a.newSessionLink(),
		a.newSessionTouch(),
		a.newSessionList(),
		a.newSessionPrune(),
	)
	return cmd
}

func (a *App) newSessionLink() *cobra.Command {
	var p session.LinkParams
	cmd := &cobra.Command{
		Use:   "link --session <id> --item <id>",
		Short: "Create or update a session's ledger record and stamp last-seen",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.Session == "" || p.Item == "" {
				return usageErr("link requires --session and --item")
			}
			bstore, err := a.backlogStore()
			if err != nil {
				return err
			}
			it, err := bstore.Find(p.Item)
			if err != nil {
				e := wrapStoreErr(err)
				e.ID = p.Item
				return e
			}
			p.Item = it.ID // the canonical key, never the raw argument
			store := session.NewStore(bstore.Root())
			rec, created, err := store.Link(p)
			if err != nil {
				return wrapSessionErr(err)
			}
			if a.json {
				return a.emitJSON(rec)
			}
			verb := "updated"
			if created {
				verb = "linked"
			}
			a.note("%s %s → %s", verb, rec.Key(), rec.Item)
			a.answer("%s", rec.Key())
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&p.Session, "session", "", "harness session id (required)")
	f.StringVar(&p.Agent, "agent", "", "subagent id inside the session")
	f.StringVar(&p.Handle, "handle", "", "claimant handle (e.g. the worker's name)")
	f.StringVar(&p.Item, "item", "", "backlog item id (required; must exist)")
	f.StringVar(&p.Role, "role", "", "user|supervisor|worker")
	f.StringVar(&p.Transcript, "transcript", "", "path to the session's transcript")
	f.StringVar(&p.Cwd, "cwd", "", "the session's working directory")
	f.StringVar(&p.Verb, "verb", "", "backlog verb behind this update: claim|deliver|close|block|release")
	return cmd
}

func (a *App) newSessionTouch() *cobra.Command {
	var sid, agent string
	cmd := &cobra.Command{
		Use:   "touch --session <id> [--agent <id>]",
		Short: "Bump a ledger record's last-seen",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sid == "" {
				return usageErr("touch requires --session")
			}
			store, err := a.sessionStore()
			if err != nil {
				return err
			}
			rec, err := store.Touch(sid, agent)
			if err != nil {
				return wrapSessionErr(err)
			}
			if a.json {
				return a.emitJSON(rec)
			}
			a.note("touched %s", rec.Key())
			a.answer("%s", rec.Key())
			return nil
		},
	}
	cmd.Flags().StringVar(&sid, "session", "", "harness session id (required)")
	cmd.Flags().StringVar(&agent, "agent", "", "subagent id inside the session")
	return cmd
}

func (a *App) newSessionList() *cobra.Command {
	var item string
	var active bool
	cmd := &cobra.Command{
		Use:   "list [--item <id>] [--active]",
		Short: "List ledger records (--active: only records whose item is in-progress or validation)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := a.resolveRoot()
			if err != nil {
				return err
			}
			recs, err := session.NewStore(root).List()
			if err != nil {
				return wrapSessionErr(err)
			}
			if item != "" {
				// Resolve any accepted reference (key, number, former id) to the
				// canonical key; an unknown reference matches ledger records
				// literally, so records of a removed item stay listable.
				if it, err := backlog.NewStore(root).Find(item); err == nil {
					item = it.ID
				} else if errors.Is(err, backlog.ErrInvalidID) {
					return wrapStoreErr(err)
				}
			}
			var activeItems map[string]bool
			if active {
				items, err := backlog.NewStore(root).LoadAll()
				if err != nil {
					return wrapStoreErr(err)
				}
				activeItems = map[string]bool{}
				for _, it := range items {
					if it.Active() {
						activeItems[it.ID] = true
					}
				}
			}
			out := []*session.Record{}
			for _, r := range recs {
				if item != "" && r.Item != item {
					continue
				}
				if active && !activeItems[r.Item] {
					continue
				}
				out = append(out, r)
			}
			if a.json {
				return a.emitJSON(out)
			}
			w := tabwriter.NewWriter(a.out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "SESSION\tAGENT\tHANDLE\tITEM\tROLE\tLAST-SEEN\tTRANSCRIPT")
			for _, r := range out {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					r.Session, dash(r.Agent), dash(r.Handle), dash(r.Item), dash(r.Role), r.Age(), dash(r.Transcript))
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&item, "item", "", "only records for this item")
	cmd.Flags().BoolVar(&active, "active", false, "only records whose item is active (in-progress or validation)")
	return cmd
}

func (a *App) newSessionPrune() *cobra.Command {
	var olderThan string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "prune [--older-than <dur|Nd>] [--dry-run]",
		Short: "Delete ledger records not seen within --older-than (default " + defaultSessionPruneAge + ")",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			age, err := parseAge(olderThan)
			if err != nil {
				return err
			}
			store, err := a.sessionStore()
			if err != nil {
				return err
			}
			res, err := store.Prune(age, dryRun)
			if err != nil {
				return wrapSessionErr(err)
			}
			if a.json {
				return a.emitJSON(res)
			}
			for _, k := range res.Pruned {
				a.answer("%s", k)
			}
			verb := "pruned"
			if dryRun {
				verb = "would prune"
			}
			a.note("%s %d session record(s), kept %d", verb, len(res.Pruned), res.Kept)
			return nil
		},
	}
	cmd.Flags().StringVar(&olderThan, "older-than", defaultSessionPruneAge, "staleness threshold (e.g. 7d, 12h)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list what would be deleted without deleting")
	return cmd
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// ledgerClaim records a claim in the sessions ledger when a harness session is
// known, so correlation works without any hook installed. A ledger failure
// never fails the claim (already committed); it is reported on stderr.
func (a *App) ledgerClaim(root string, it *backlog.Item) {
	if it.Session == "" {
		return
	}
	cwd, _ := os.Getwd()
	_, _, err := session.NewStore(root).Link(session.LinkParams{
		Session: it.Session,
		Handle:  it.ClaimedBy,
		Item:    it.ID,
		Role:    backlog.ResolveReporter(""),
		Cwd:     cwd,
		Verb:    "claim",
	})
	if err != nil {
		a.note("warning: sessions ledger not updated: %v", err)
	}
}

// ledgerTouch bumps the ledger records of the caller's session that name the
// item, if any exist. The session is CLAUDE_CODE_SESSION_ID, else the item's
// stamped session. Failures are reported on stderr, never fatal.
func (a *App) ledgerTouch(root, itemID, itemSession, verb string) {
	sid := backlog.ResolveSession("")
	if sid == "" {
		sid = itemSession
	}
	if sid == "" {
		return
	}
	if _, err := session.NewStore(root).TouchItem(sid, itemID, verb); err != nil {
		a.note("warning: sessions ledger not updated: %v", err)
	}
}
