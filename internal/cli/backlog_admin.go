package cli

import (
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
)

func (a *App) newBacklogMigrate() *cobra.Command {
	var dryRun bool
	var defaultReporter, keyPrefix string
	cmd := &cobra.Command{
		Use:   "migrate [--key-prefix XXX]",
		Short: "Upgrade the backlog to schema 3 (PREFIX-N item keys) and stamp workstreams.md (idempotent)",
		Long: "migrate maps schema-1 fields to schema 2, then keys every item still filed under its slug:\n" +
			"numbers in created-at order, files renamed to NNNN-<slug>.md, former-id recorded, sessions-ledger\n" +
			"and escalation references rewritten, sequence and schema: 3 + key-prefix stamped. --key-prefix is\n" +
			"required when workstreams.md carries no key-prefix; otherwise it must match.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkEnumFlag("reporter", defaultReporter); err != nil {
				return usageErr("--default-reporter must be one of " + enumList("reporter"))
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			if keyPrefix != "" {
				keyPrefix = strings.ToUpper(strings.TrimSpace(keyPrefix))
				if !backlog.ValidPrefix(keyPrefix) {
					return usageErr("--key-prefix must be 2 to 5 letters (e.g. ANT)")
				}
				if have, perr := store.KeyPrefix(); perr == nil && have != "" && have != keyPrefix {
					return usageErr("--key-prefix " + keyPrefix + " does not match the installation's key-prefix " + have)
				}
			}
			res, err := store.Migrate(backlog.MigrateOptions{DryRun: dryRun, DefaultReporter: defaultReporter, KeyPrefix: keyPrefix})
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(res)
			}
			verb := "migrated"
			if dryRun {
				verb = "would migrate"
			}
			for _, c := range res.Changed {
				if c.FormerID != "" {
					a.answer("%s → %s", c.FormerID, c.ID)
					continue
				}
				a.answer("%s: %s", c.ID, strings.Join(c.Changes, "; "))
			}
			for _, r := range res.References {
				a.answer("rewrite %s", r)
			}
			for _, s := range res.Skipped {
				a.answer("skipped %s: %s", s.Path, s.Reason)
			}
			stamp := ""
			if res.SchemaStamped || res.PrefixStamped {
				var keys []string
				if res.SchemaStamped {
					keys = append(keys, "schema: 3")
				}
				if res.PrefixStamped {
					keys = append(keys, "key-prefix: "+res.KeyPrefix)
				}
				stamp = "; " + strings.Join(keys, ", ") + " stamped into workstreams.md"
				if dryRun {
					stamp = "; would stamp " + strings.Join(keys, ", ") + " into workstreams.md"
				}
			}
			a.note("%s %d item(s), %d already on schema 3, %d skipped%s",
				verb, len(res.Changed), res.Unchanged, len(res.Skipped), stamp)
			if len(res.Skipped) > 0 {
				a.note("skipped files are untouched; fix them by hand, then run 'anthill backlog validate'")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report the changes without writing")
	cmd.Flags().StringVar(&defaultReporter, "default-reporter", "user", "reporter for items that lack one")
	cmd.Flags().StringVar(&keyPrefix, "key-prefix", "", "item key prefix to stamp (2-5 letters; required when workstreams.md has none)")
	return cmd
}

func (a *App) newBacklogRenumber() *cobra.Command {
	var to int
	var by string
	cmd := &cobra.Command{
		Use:   "renumber <id> [--to N]",
		Short: "Move an item to another key number (repairs a branch-merge collision)",
		Long: "renumber renames the item file to the new number (default: the next free one), raises sequence,\n" +
			"rewrites every sessions-ledger and escalation record naming the old key, and logs the move.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("to") && to < 1 {
				return usageErr("--to must be a positive number")
			}
			store, it, err := a.findItem(args[0])
			if err != nil {
				return err
			}
			res, err := store.Renumber(it, to, by)
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(res)
			}
			a.note("renumbered %s → %s", res.From, res.To)
			for _, r := range res.References {
				a.note("rewrote %s", r)
			}
			a.answer("%s", res.To)
			return nil
		},
	}
	cmd.Flags().IntVar(&to, "to", 0, "target number (default: next free)")
	cmd.Flags().StringVar(&by, "by", "", "actor handle for the Log line (default: $ANTHILL_ACTOR, else role:session, else OS user)")
	return cmd
}

// parseAge parses a Go duration, also accepting a whole number of days ("30d").
func parseAge(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil && n >= 0 {
			return time.Duration(n) * 24 * time.Hour, nil
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, usageErr("--older-than must be a duration like 30d or 72h")
	}
	return d, nil
}

func (a *App) newBacklogPrune() *cobra.Command {
	var olderThan string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Delete discarded items from discarded/ (manual; done/ is never pruned)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var age time.Duration
			if olderThan != "" {
				d, err := parseAge(olderThan)
				if err != nil {
					return err
				}
				age = d
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			res, err := store.Prune(age, dryRun)
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(res)
			}
			for _, id := range res.Pruned {
				a.answer("%s", id)
			}
			verb := "pruned"
			if dryRun {
				verb = "would prune"
			}
			a.note("%s %d discarded item(s), kept %d", verb, len(res.Pruned), res.Kept)
			return nil
		},
	}
	cmd.Flags().StringVar(&olderThan, "older-than", "", "only items closed longer ago than this (e.g. 30d, 72h)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list what would be deleted without deleting")
	return cmd
}
