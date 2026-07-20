# Escalations

Durable decision requests traveling up the agent tiers
(worker → dispatcher → supervisor → user). Full contract: the `escalate`
skill. One file per open escalation; closed ones are one-lined in `LOG.md`
and deleted.

- The `to:` field — not the signal path — determines who owns a record.
- Every controller tier sweeps this directory on every wake-up.
- The originating Question travels verbatim; tiers append, never rewrite.
- `answered` records are actionable by whoever finds them (apply → LOG.md).

Template (`<yyyy-mm-dd>-<slug>.md`):

    ---
    to: supervisor            # dispatcher | supervisor | user
    from: dispatcher
    item: <backlog id>        # optional — omit for non-item incidents
    status: open              # open | answered | applied
    opened: <yyyy-mm-dd>
    ---
    ## Question
    (verbatim from the originating agent)
    ## Context & attempted remedies
    (dated blocks, append-only, one per tier that raises)
    ## Options & recommendation
    (background, the catch, options contrasted, a recommendation with the reason)
    ## Decision
    (appended by the answerer; sets status: answered)
    ## Applied
    (appended by the applier; then one-line to LOG.md and delete)
