# Worker brief template

Fill every section; delete nothing. The first line of the actual brief is
always the autonomy-contract invocation.

---

/autonomous

GOAL
<one paragraph of intent, then an ordered task list with any
user-specified sequencing>

DEFINITION OF DONE
<observable outcomes. NAME THE EVIDENCE to attach per deliverable —
the supervisor verifies evidence, not assertions. Mandatory evidence per
deliverable class: .anthill/supervisor/bindings.md>

TERRITORY
Yours: <files/folders>
Off-limits: <the other worker's territory, named>
Shared resources: <rules per .anthill/supervisor/bindings.md — VCS
discipline, resource turn-taking>

REPORTING
Reply to: <the exact message target, written by the spawner — the
registered agent name or the raw agentId from the spawn result. Never a
bare role word like "the supervisor"; if no handle resolves, report in
your final message>
Land state before signaling: put results somewhere durable first (commit,
item body, escalation record), then message — a lost message must cost
latency, never content. Report on material transitions only: blocked /
needs-decision / done / failed, one line each. No chatter. Before
reporting blocked, consider whether the blocker is the other worker's
in-progress work — report that as co-tenancy friction instead.
