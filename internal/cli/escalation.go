package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/escalation"
)

// newEscalationCommand builds the `anthill escalation` group.
func (a *App) newEscalationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "escalation",
		Aliases: []string{"esc"},
		Short:   "Raise, list, answer, and apply escalation records",
	}
	cmd.AddCommand(
		a.newEscalationRaise(),
		a.newEscalationList(),
		a.newEscalationShow(),
		a.newEscalationAnswer(),
		a.newEscalationApply(),
	)
	return cmd
}

func (a *App) newEscalationRaise() *cobra.Command {
	var to, from, item, question, context, options, bodyFile string
	cmd := &cobra.Command{
		Use:   "raise",
		Short: "Create a durable escalation record",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.escalationStore()
			if err != nil {
				return err
			}
			r, err := store.Raise(escalation.RaiseParams{
				To: to, From: from, Item: item, Question: question,
				Context: context, Options: options, BodyFile: bodyFile,
			})
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(viewRecord(r))
			}
			a.note("raised %s (to: %s)", r.ID, r.To)
			a.answer("%s", r.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "dispatcher|supervisor|user (required)")
	cmd.Flags().StringVar(&from, "from", "", "originating tier (required)")
	cmd.Flags().StringVar(&item, "item", "", "related backlog id (optional)")
	cmd.Flags().StringVar(&question, "question", "", "the verbatim question (required)")
	cmd.Flags().StringVar(&context, "context", "", "Context & attempted remedies body")
	cmd.Flags().StringVar(&options, "options", "", "Options & recommendation body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "supply the full markdown body from a file")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("question")
	return cmd
}

func (a *App) newEscalationList() *cobra.Command {
	var to, status, item string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Sweep the escalation directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.escalationStore()
			if err != nil {
				return err
			}
			recs, err := store.LoadAll()
			if err != nil {
				return internalErr(err.Error())
			}
			var filtered []*escalation.Record
			for _, r := range recs {
				if to != "" && r.To != to {
					continue
				}
				if status != "" && r.Status != status {
					continue
				}
				if item != "" && r.Item != item {
					continue
				}
				filtered = append(filtered, r)
			}
			if a.json {
				return a.emitJSON(viewRecords(filtered))
			}
			if len(filtered) == 0 {
				a.note("no records match")
				return nil
			}
			a.printRecordTable(filtered)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "records addressed to a tier")
	cmd.Flags().StringVar(&status, "status", "", "open|answered|applied")
	cmd.Flags().StringVar(&item, "item", "", "records tied to a backlog id")
	return cmd
}

func (a *App) newEscalationShow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Print one full escalation record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.escalationStore()
			if err != nil {
				return err
			}
			r, err := store.Find(args[0])
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(viewRecord(r))
			}
			a.printRecordDetail(r)
			return nil
		},
	}
	return cmd
}

func (a *App) newEscalationAnswer() *cobra.Command {
	var decision string
	cmd := &cobra.Command{
		Use:   "answer <id>",
		Short: "Record a decision on an open record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if decision == "" {
				return usageErr("--decision is required")
			}
			store, err := a.escalationStore()
			if err != nil {
				return err
			}
			r, err := store.Answer(args[0], decision)
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(viewRecord(r))
			}
			a.note("answered %s", r.ID)
			a.answer("%s", r.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&decision, "decision", "", "the decision (appended as ## Decision, required)")
	return cmd
}

func (a *App) newEscalationApply() *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:   "apply <id>",
		Short: "Close out an answered record: mark applied, log, delete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.escalationStore()
			if err != nil {
				return err
			}
			r, err := store.Apply(args[0], note)
			if err != nil {
				return wrapStoreErr(err)
			}
			if a.json {
				return a.emitJSON(map[string]any{"id": r.ID, "applied": true, "logged": true})
			}
			a.note("applied %s", r.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "one-line outcome appended under ## Applied")
	return cmd
}

// printRecordDetail renders one record to stdout.
func (a *App) printRecordDetail(r *escalation.Record) {
	fmt.Fprintf(a.out, "id:      %s\n", r.ID)
	fmt.Fprintf(a.out, "to:      %s\n", r.To)
	fmt.Fprintf(a.out, "from:    %s\n", r.From)
	if r.Item != "" {
		fmt.Fprintf(a.out, "item:    %s\n", r.Item)
	}
	fmt.Fprintf(a.out, "status:  %s\n", r.Status)
	fmt.Fprintf(a.out, "opened:  %s\n", r.Opened)
	for _, name := range escalation.SectionOrder {
		if body := r.Section(name); body != "" {
			fmt.Fprintf(a.out, "\n## %s\n%s\n", name, body)
		}
	}
}
