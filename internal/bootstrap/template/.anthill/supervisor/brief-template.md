# Worker brief template

Fill every section; delete nothing. The first line of the actual brief is
always the autonomy-contract invocation.

---

/autonomous

GOAL
<the ONE task this worker carries: one result (a change, a report, a
decision), one paragraph of intent, then the ordered steps toward it
with any user-specified sequencing. Work drawing on unrelated
information goes to its own worker>

BATCH REASON
<always present. "none" for a one-task brief. A brief with several parts
names the artifacts they share (files, documents, logs, backlog items,
records) and the single result they feed, e.g. "all steps read the open
backlog and the playtest logs and feed one triage report; split, each
worker would re-read both." The supervisor's delegation gate rejects a
brief whose parts draw on unrelated information>

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
