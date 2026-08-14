---
sweep-order: bugs, product, dev, process
never-implicit:
---

# Backlog workstreams — <YOUR PROJECT>

> **PROJECT-SPECIFIC TIER — TEMPLATE.** This ships the four framework-default
> workstreams (rename `product` to your product's name — in the frontmatter,
> the section heading below, AND the directory `.anthill/backlog/product/`).
> Add your own project-specific workstreams in the marked slot. Tune each
> triage profile to your domain. Delete this quote block once derived.

The project's workstream definitions: what belongs in each, how its items are
triaged (profile), how they get implemented (dispatch route), and what evidence
closes them. Loaded by the `triage`/`dispatch`/`expedite` skills on invocation.

The frontmatter above is the machine-readable contract:
- **sweep-order** — the order bare `dispatch` walks workstreams.
- **never-implicit** — workstreams only ever dispatched deliberately (never
  swept by bare `dispatch`). Leave empty unless you have a stream that must
  never auto-run (e.g. a content/authoring stream); then list it here.

A workstream's identity is its **directory name** under
`.anthill/backlog/`. Every stream below must have a matching directory.

---

## product

> Rename to your product's name everywhere (frontmatter, this heading,
> the directory). This is the thing the project exists to build.

The product: <the capabilities of the thing you're building — its features,
defaults, semantics, surfaces>.

- **Triage profile:** improvement gates —
  - *Value gate:* benefit ÷ permanent cognitive cost (what the change adds
    to the surface everyone must learn — not implementation effort).
    Heuristics: recurring not one-off · distinct not redundant · painful
    workaround · composable · smallest footprint that delivers it. Prefer
    the cheapest change type that works.
  - *Safety gates* (for an AUTO recommendation): additive-or-reversible ·
    verifiable (a concrete `verify`) · bounded scope · unambiguous spec.
  - *<Any domain-specific pre-checks — e.g. a dedup check against an
    existing catalog before adding a new primitive>.*
- **Never-auto:** <the change types that are permanent, cross-cutting, or
  taste-laden in your domain — they cap at human review no matter how safe.
  e.g. adding a new first-class primitive/surface>.
- **Dispatch route:** `dispatch` skill; <point specialized item types at
  their authoring skill if you have one>.
- **Evidence:** <your build/verify command> exit 0; the item's `verify` test.

## dev

Development-process tooling: the CLI(s), test harnesses, bridges, command
servers, compile/profiling tools, diagnostics — everything that speeds the
development loop itself.

- **Triage profile:** improvement gates as for `product`, minus any
  product-specific pre-checks. Weight tooling value by how much it unblocks
  the agentic loop.
- **Never-auto:** changes to safety invariants or permission surfaces.
- **Dispatch route:** `dispatch` skill.
- **Evidence:** the headless test / exit code named in `verify`.

## process

How information and work flow through the project: docs, playbooks,
codification, backlog and Anthill configuration. Changes to Anthill
*mechanisms* themselves are rare and get flagged upstream to the framework
home (see `.anthill/framework.md`) rather than patched locally — local
divergence across installations is the failure mode to avoid.

- **Triage profile:** improvement gates; plus the instruction-file rule —
  *reference material* goes to a scoped home that loads only when relevant;
  only a *standing behavioral directive* earns a place in an always-on file
  (CLAUDE.md), and the bar is high.
- **Never-auto:** edits to always-on instruction files (CLAUDE.md).
- **Dispatch route:** `dispatch` skill (mostly `doc` change-type). A learning
  lands in its one durable home — never a second home for a fact that has one.
- **Evidence:** the doc/codification exists in its durable home and nothing
  else claims to own the same fact.

## bugs

Defects in intended existing behavior, regardless of component. Routing rule:
restore-intended-behavior → here; capability/improvement work → the
component's workstream.

- **Triage profile:** light — the value gate auto-passes (correctness is its
  own value). Requires a reproduction and a headless `verify`.
- **Never-auto:** behavior changes without a regression guard.
- **Dispatch route:** `dispatch` skill. Default `priority: high`.
- **Evidence:** the repro fails before the fix and passes after;
  <your build/verify command> exit 0.

<!-- PROJECT-SPECIFIC WORKSTREAM SLOT ---------------------------------------
Add streams your project needs beyond the four defaults. A common one is a
"content"/authoring stream for the artifacts the project produces (as opposed
to the tools that produce them). If a stream should never be swept
automatically, add its name to `never-implicit` in the frontmatter.

Example shape:

  ## content
  The authoring pipeline — the artifacts the project exists to produce
  (the project dogfooding itself). Items are authoring asks, not tool
  improvements; a tool gap found while authoring is extracted to
  product/dev/bugs as its own item.
  - Triage profile: feasibility (do the needed capabilities exist?) +
    direction fit. Taste decisions surface to the user, never auto-resolved.
  - Never-auto: everything — only ever dispatched deliberately (listed in
    `never-implicit`).
  - Dispatch route: <your authoring skill>, explicitly assigned.
  - Evidence: <the artifact + a render/report proving it>.
------------------------------------------------------------------------- -->

---

## Judgment signals (accrued from triage decisions — read before classifying)

> Starts empty. When a triage decision generalizes into a reusable rule,
> append it here so the next triage inherits it. Examples of the kind of
> signal that accrues (from the Nodachi template):

- **Re-run dedup after approvals.** Approving one capability can make a
  pending item redundant.
- **A narrow fix in-flight is NOT a reason to skip the general primitive.**
  Surface "narrow vs general" as a user choice.
- **Don't force a band-aid around a footgun.** If every disposition is a
  cosmetic patch on a root-cause design smell, escalate to
  needs-investigation instead of picking the least-bad patch.
- **Consolidate on intake; split framing items.** Fold narrow items under an
  umbrella; split a bundle into approved high-leverage pieces + a parked
  remainder.
