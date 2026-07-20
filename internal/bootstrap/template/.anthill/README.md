# `.anthill/` — the project's harness adaptation layer

Everything the Anthill harness produces or consumes lives here, checked in
with the project. This is the **specific tier**: the general-tier mechanisms
live in `.claude/skills/` (portable verbatim) and load their config from
this directory on invocation.

Every file here is one of two classes:

- **Configuration** (authored with the user, stable) — the project's
  bindings, templates, taxonomies, resource inventory. These are the files
  you *derive* when adopting Anthill.
- **Runtime** (agent-written during operation) — live harness state: the
  supervisor agenda, backlog items, escalation records, decision log.

## Map

```
.anthill/
  README.md            # this file
  framework.md         # config  — provenance + sync state against the framework home
  resources.md         # config  — exclusive-resource inventory + derived caps
  decisions.md         # runtime — routine-choice log for autonomous work
  supervisor/
    bindings.md        # config  — worker cap pointer, evidence rules, silence
                       #           threshold, real skill/command names, intake, launcher
    brief-template.md  # config  — worker brief skeleton (portable; adjust paths only)
    agenda.md          # runtime — user intent ONLY; rehydration source
    scratchpad/        # runtime — gitignored investigation reasoning traces
  backlog/
    workstreams.md     # config  — workstream defs + triage profiles + sweep order
    bindings.md        # config  — schema owner, commands, id scheme, posture
    README.md          # config  — how to submit (intake)
    intake/            # runtime — untriaged, workstream-less items
    product/ dev/ process/ bugs/   # runtime — open items per workstream
    CHANGELOG.md       # runtime — one line per closed item
  escalations/
    README.md          # config  — record template (portable)
    LOG.md             # runtime — one line per closed escalation
```

## Adoption rule

Take the `.claude/skills/` general tier unchanged; **derive** this directory
from your environment. Runtime artifacts start empty (the agenda seeds from
the user's first briefing). See `INSTALLATION.md` at the repo root.
