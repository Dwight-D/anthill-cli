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
	var defaultReporter string
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Upgrade every backlog item to schema 2 and stamp workstreams.md (idempotent)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkEnumFlag("reporter", defaultReporter); err != nil {
				return usageErr("--default-reporter must be one of " + enumList("reporter"))
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			res, err := store.Migrate(backlog.MigrateOptions{DryRun: dryRun, DefaultReporter: defaultReporter})
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
				a.answer("%s: %s", c.ID, strings.Join(c.Changes, "; "))
			}
			for _, s := range res.Skipped {
				a.answer("skipped %s: %s", s.Path, s.Reason)
			}
			stamp := ""
			if res.SchemaStamped {
				stamp = "; schema: 2 stamped into workstreams.md"
				if dryRun {
					stamp = "; would stamp schema: 2 into workstreams.md"
				}
			}
			a.note("%s %d item(s), %d already on schema 2, %d skipped%s",
				verb, len(res.Changed), res.Unchanged, len(res.Skipped), stamp)
			if len(res.Skipped) > 0 {
				a.note("skipped files are untouched; fix them by hand, then run 'anthill backlog validate'")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report the changes without writing")
	cmd.Flags().StringVar(&defaultReporter, "default-reporter", "user", "reporter for items that lack one")
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
