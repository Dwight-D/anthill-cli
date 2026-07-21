# Dispatch progress ledger

Updated: <date> (<who / session>)

> **RUNTIME ARTIFACT — seed on first use.** The dispatcher's durable position
> in a framed batch: the ordered item list, per-item status, and the framing
> (who to report to, the reporting cadence, the batch's origin/authorization).
> It holds POSITION only — never findings, never an item's content (that lives
> in the backlog item). A cold successor rehydrates its cursor from here; the
> cursor is the first `pending` row. Present only while a framed batch is in
> flight; a bare ready-queue sweep needs no ledger. Delete or archive when the
> batch closes. The placeholder below shows the shape.

## Batch framing

- **Origin / authorization:** <who framed this batch and when — e.g. supervisor
  brief 2026-07-20, or user directive>
- **Report-target:** <the tier to report to — e.g. supervisor / user>
- **Reporting cadence:** <e.g. after each item · on batch completion · on
  escalation only>

## Items (in order)

| # | item id | status | outcome / pointer |
|---|---------|--------|-------------------|
| 1 | <id>    | done \| in-progress \| pending \| blocked \| escalated | <one line: changelog ref, block reason, or escalation file> |
| 2 | <id>    | pending | |
| … | …       | …       | |
