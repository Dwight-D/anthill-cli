# Supervisor scratchpad

Ephemeral reasoning traces from investigation/analysis subagents the
supervisor spawns. The supervisor consumes only the conclusion; the trail
of HOW the conclusion was reached lands here so the user can follow the
reasoning later or troubleshoot a false conclusion.

- One file per investigation: `<yyyy-mm-dd>-<topic>.md`.
- Written by the investigating subagent itself, before it returns its
  conclusion; the conclusion message includes the file path.
- Content: what was examined, the evidence found, how the conclusion
  follows. Raw and unpolished is fine — this is a trace, not a report.
- Ephemeral: not committed (gitignored), prune freely once stale.
