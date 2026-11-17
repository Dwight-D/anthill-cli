# Item keys — `PREFIX-N` ids, file-based allocation, and the in-session plugin surface

Status: approved contract (user, 2026-11-17). Supersedes the id rules of
`BACKLOG_V2_SPEC.md` §1.1 (title row), §1.3 and §6.1, and of
`CLI_INTERFACE_SPEC.md` §4 "Id generation"; those sections are updated to
point here as part of phase 1. Everything else in `BACKLOG_V2_SPEC.md`
stands. Schema version 3.

Three phases, three repos, in order: (1) `anthill-cli`, (2) `anthill-fw`,
(3) `anthill-claude`. Each phase section is self-contained for its worker.

---

## 1. The key

A backlog item's id is a **key**: an installation prefix, a hyphen, and a
positive integer with no padding. Examples: `ANT-74`, `BH-3`.

- **Prefix**: 2 to 5 uppercase ASCII letters, set once per installation as
  `key-prefix: ANT` in the frontmatter of `.anthill/backlog/workstreams.md`.
  `anthill init` seeds it (`--key-prefix`, default: the first three letters
  of the root directory's name, uppercased, letters only; fewer than two
  letters → `ANT`). Changing the prefix later is a config edit that leaves
  every number in place; it is discouraged because prose, ledger records and
  escalations written under the old prefix stop resolving.
- **Number**: allocated sequentially per installation (§2). Never reused.
- **Canonical form** is what the CLI prints everywhere (`id` in JSON and
  tables, `created ANT-74`, Log lines, the sessions ledger `item`,
  `AppendChangelog`).
- **Accepted input** wherever a verb takes an id (`show`, `set`, `claim`,
  `deliver`, …, `session link --item`, `escalation raise --item`,
  `list --item`):
  1. the canonical key, case-insensitive (`ant-74`);
  2. the bare number (`74`), resolved with the installation's prefix;
  3. a former id (§4): the pre-key slug recorded in the item's `former-id`
     field. Matched exactly, lowercase.
  Anything else (empty, path separators, `..`, other characters) exits 2
  (usage) with `invalid item id "<input>": expected ANT-74, 74, or a
  former id`. `Find` never joins unparsed input into a path.
- **Regex** (for tools; the CLI uses a parser): canonical
  `^[A-Z]{2,5}-[1-9][0-9]*$`; former id `^[a-z0-9][a-z0-9-]{0,49}$`.
- **Ordering**: keys compare by number. `--sort id`, the default `rank`
  tie-break, and `Load`'s ordering all compare numerically (`ANT-74` before
  `ANT-100`). Equal numbers never occur in a valid tree.

## 2. Files and allocation

### 2.1 Layout

```
.anthill/backlog/
  workstreams.md          frontmatter: schema: 3, key-prefix: ANT, …
  sequence                one line: the highest number ever allocated
  intake/0074-anthill-claude-now-pane.md
  cli/0066-escalation-answer-file.md
  done/0012-first-release.md
  discarded/0009-dup-of-eight.md
```

- Filename: `NNNN-<slug>.md`. `NNNN` is the number zero-padded to at least
  four digits (`0074`, `0100`, `12345`). `<slug>` is the title slug of today's
  `Slug()` (lowercase kebab, ≤ 40 characters after truncation at a dash
  boundary), cosmetic only; an empty slug writes `NNNN-item.md`.
- The id is parsed from the filename: the leading digits give the number;
  the prefix comes from `workstreams.md`. The slug tail is ignored on read.
  `set title=…` renames the file to the new slug (same number) and appends a
  Log line; nothing else references the tail.
- A file in an item directory that does not match `^\d{4,}-[a-z0-9-]+\.md$`
  is a validate `stray-file` violation (as a non-`.md` file is today).
- `sequence` is a plain text file holding one integer and a newline. It is
  a high-water mark, never the authority: it protects numbers freed by
  `close --remove` and `prune` from reuse. It is tracked in git like the
  items are. Missing or unparsable → treated as 0.

### 2.2 Allocation (`new`)

1. `next = max(highest number parsed from every filename in intake/, each
   workstream dir, done/, discarded/; the integer in sequence) + 1`.
2. Create `intake/NNNN-<slug>.md` with `os.OpenFile(path,
   O_WRONLY|O_CREATE|O_EXCL, 0o644)` and write the item content directly.
   The filesystem create is atomic on NTFS and POSIX, so two concurrent
   `new` calls cannot share a number. On `EEXIST`, or if any file
   `NNNN-*.md` already exists in any item directory, rescan and retry;
   after 20 attempts exit 5 (conflict).
3. Write `sequence` = the number just used, via the existing
   `mdfile.WriteAtomic`, only if it is higher than the stored value.
   A failure here is a warning, never a failed `new`.
4. Print `created ANT-74`; under `--json`, the item view with `"id":
   "ANT-74"`.

The two-branch hole (two git branches each allocate `ANT-75`, then merge)
is detected by `validate`/`doctor` (`id-unique` across directories) and
repaired by `renumber` (§3.3).

## 3. CLI changes (phase 1, `anthill-cli`, schema 3)

### 3.1 Resolution

- `backlog.ParseKey(input string, prefix string) (Key, error)`: the
  accepted-input rules of §1. `Key{Prefix, Number}` with `String()`
  (`ANT-74`), `FileStem()` (`0074`), numeric `Less`.
- `Store.Find(input)`: parse; on a key, glob `NNNN-*.md` in intake, each
  workstream dir, done, discarded (first match wins; a second match in
  another directory is reported by validate, not by Find); on a former id,
  `Load(true)` and match `former-id`. Not found → exit 4 as today.
- Every site that echoes the user's argument (`set`, `close`, the
  approve/fast-track/unapprove error text, `session link`) prints and
  stores `it.ID` (canonical), never the raw input.
- `Item.ID` stays unserialised (`yaml:"-"`); `former-id` is a new optional
  frontmatter key (`former-id: <slug>`, string, written only by `migrate`,
  never by `set`; `validate` knows it).

### 3.2 workstreams frontmatter

- New key `key-prefix` (string). `SchemaVersion` gains a sibling
  `KeyPrefix() (string, error)`: missing while items exist → `validate`
  violation `key-prefix-missing` and `new`/`migrate` exit 6 with the fix
  (`add key-prefix: XXX to .anthill/backlog/workstreams.md or run migrate
  --key-prefix XXX`); malformed → `key-prefix-invalid`.
- `stampFrontmatter` is generalised to set any scalar key
  (`stampFrontmatterKey(key, value)`); `StampSchema` uses it with
  `CurrentSchema = 3`.
- `init` writes `schema: 3` and `key-prefix: <derived or --key-prefix>`.
  The embedded template's `workstreams.md` is updated upstream in phase 2;
  until the re-vendor, `init`'s own writer (not the template copy) must
  produce a valid tree, and `scaffold`/`bootstrap` output is stamped by the
  first `new` as today (schema 0 → stamp both keys).

### 3.3 Verbs

- `anthill backlog renumber <id> [--to N]`: moves the item to `N` (default:
  next free number per §2.2), renames the file, bumps `sequence`, rewrites
  the `item` of every sessions-ledger record and every escalation record
  that names the old key, appends a Log line `renumber ANT-75 → ANT-81 by
  <actor>`. Exit 5 if `N` is taken. Needed only after a branch-merge
  collision.
- `anthill backlog migrate` performs v2 → v3 (§4) after v1 → v2 when the
  tree is at 1; idempotent at 3. New flag `--key-prefix XXX` (required when
  the frontmatter has none; otherwise it must match or exit 2).
- `validate` / `doctor` rules added: `key-prefix-missing`,
  `key-prefix-invalid`, `filename-form` (a `.md` not matching the §2.1
  pattern), `id-unique` now keyed on the number across all directories,
  `former-id-unique`. Message text that hard-codes "schema 2" says 3.
- `list --fields` and JSON gain no new field: `id` is the key. `title` stays
  the human label; the TUI Backlog mode adds a title column after the id so
  rows stay readable (`backlogmode.go`); Now mode rows show the key.
- `next`, `claim`, `deliver`, `close` etc. keep their output shapes with the
  key in place of the slug.

### 3.4 Sessions ledger and escalations

- `session link --item` resolves through `Find` and stores the canonical
  key. `list --item` resolves its argument the same way before matching.
- `escalation raise --item` resolves through the backlog `Find` when a
  backlog store exists at the root and stores the canonical key; an
  unresolvable item exits 4. Escalation ids themselves are unchanged in
  this phase.

### 3.5 Migration v2 → v3

In `migrate`, after any v1 → v2 pass, when `schema` is 2 (or the tree is
unstamped and items exist in the old `<slug>.md` form):

1. Determine the prefix: frontmatter `key-prefix`, else `--key-prefix`,
   else exit 6 with the message of §3.2.
2. Load every item from intake, each workstream dir, done, discarded. Order
   by `created-at` ascending, then by old id ascending. Assign numbers 1…n
   in that order.
3. For each item: rename `<slug>.md` → `NNNN-<slug>.md` in place (same
   directory; slug truncated to 40), write `former-id: <slug>`, append Log
   `migrate schema 2 → 3: <slug> → ANT-N by anthill`.
4. Write `sequence` = n. Stamp `schema: 3` and `key-prefix`.
5. Rewrite references: every `.anthill/sessions/*.json` `item` equal to an
   old id → its key; every escalation record whose `item:` equals an old id
   → its key (the escalation file is otherwise untouched). `CHANGELOG.md`
   lines are left as they are (history).
6. `--dry-run` prints `old-id → ANT-N` per item and the reference rewrites
   it would make. `MigrateChange` gains `FormerID`.
7. The pass is idempotent: an item already in `NNNN-` form with the current
   prefix is skipped; a tree at schema 3 reports nothing.

Consumer action after upgrading the CLI: `anthill backlog migrate
[--key-prefix XXX]`, then `anthill sync` for the skills.

### 3.6 Tests (evidence for phase 1)

- Unit: `ParseKey` table (canonical, lowercase, bare number, former id,
  rejects `../x`, `ANT-0`, `ANT-074`, `A-1`, `ABCDEF-1`); filename parse
  table; numeric sort; `stampFrontmatterKey` insert and replace.
- e2e (the harness in `test/`): `itemPath`/`writeItem` helpers take a
  number and a slug and write `NNNN-<slug>.md`; `mkTree` writes `schema: 3`
  and `key-prefix: ANT`. `new` allocates `ANT-1`, `ANT-2`; after
  `close --remove ANT-2` the next is `ANT-3` (sequence); `show 2`,
  `show ant-2`, `show <former-id>` all resolve; `set title=` renames the
  file; `renumber` moves and rewrites a ledger record and an escalation;
  parallel allocation: 8 goroutines (or 8 concurrent `new` processes)
  produce 8 distinct keys; migrate fixture `test/testdata/migrate-v2/`
  built from this installation's schema-2 items (copy them from
  `../.anthill/backlog/`), asserting the numbering order, `former-id`,
  `sequence`, and idempotency; `migrate-v1` fixture now ends at schema 3;
  validate reports a duplicate number across directories and a stray
  filename.
- `go build ./...` and `go test ./...` exit 0 in `anthill-cli/`.
- `docs/BACKLOG_V2_SPEC.md` §1.1 title row, §1.3, §6.1, §6.6, §9 and
  `docs/CLI_INTERFACE_SPEC.md` §4 updated to state the key scheme or point
  here; `CHANGELOG.md` entry with the consumer action; `CLAUDE.md` (repo)
  unchanged. CLI version bump is the release skill's job, not the worker's.

## 4. Framework changes (phase 2, `anthill-fw`)

The framework describes items by key. Every artifact must read
self-contained (no "formerly", no "instead of slugs" outside the
CHANGELOG):

- `.anthill/backlog/bindings.md`: the "Id scheme" section states: id =
  key `PREFIX-N`, prefix in `workstreams.md` (`key-prefix`), numbers
  allocated by the CLI and never reused, filename `NNNN-<slug>.md`, verbs
  accept the key, the bare number, or a former id. The `title` row says the
  slug in the filename is cosmetic. Verb examples use `ANT-74`.
- `.anthill/backlog/workstreams.md` template: frontmatter `schema: 3` and
  `key-prefix: {{PREFIX}}` (the template's placeholder character and
  convention as used elsewhere in the template; document that `anthill init`
  fills it).
- `.anthill/backlog/README.md`, `INSTALLATION.md` step 6, the
  `CLAUDE.template.md` wording of the "id scheme" invariant: keys.
- Skills (`submit`, `triage`, `dispatch`, `dispatch-loop`,
  `dispatch-receive`, `supervisor`, `autonomous`, `escalate`, `expedite`):
  wherever they say "id", examples show a key; `submit` reports "the created
  key (e.g. `ANT-74`)"; the brief template's ITEM section shows `ANT-74`;
  `dispatch/ledger.md` column header "key".
- A short "Referring to items" paragraph in `bindings.md`: write the key in
  prose exactly as the CLI prints it (`ANT-74`), so humans and tools
  recognise it; the Claude Code plugin draws a chip for every key in a
  transcript row.
- `CHANGELOG.md`: entry "Item keys" with the consumer action (`anthill
  backlog migrate --key-prefix XXX`, then `anthill sync`).
- Evidence: `grep -rn "<id>\.md\|kebab slug" anthill-fw/` finds nothing
  that states the slug scheme as current; every edited skill reads
  self-contained; CHANGELOG entry present.

After phase 2 lands: re-vendor the template into the CLI (ref bump),
`anthill sync` this installation, `anthill backlog migrate --key-prefix ANT`
here and `--key-prefix BH` in battlehounds (user runs the latter or asks).

## 5. Plugin surface (phase 3, `anthill-claude` 0.4.0)

One new feature module `hooks/features/now-pane.tsx` registered from
`hooks/register.tsx` beside the existing three, plus a pure helper
`hooks/features/now-pane/keys.ts` (key regex, extraction from text) and
`hooks/features/now-pane/model.ts` (JSON → row model) with tests under
`tests/`. Mod API names below are verified in Claude Code 2.1.318
(`plugin-authoring` skill, `types/claude-code.d.ts`).

### 5.1 Installation discovery

On `session.start`: if `<cwd>/.anthill/backlog/workstreams.md` exists
(`$.fs`), read its `key-prefix:` line; store `{ root, prefix }` in
`$.state` (declared in `types/index.d.ts` under the plugin's name). No
installation → the feature is inert for the session (no pane, no chips,
no status).

### 5.2 Data

- `refresh()` runs `$.process.run(['anthill','backlog','list','--active',
  '--json'], { cwd: root, timeoutMs: 10_000 })` and
  `['anthill','session','list','--json']`, parses both, builds rows
  (`key, title, summary, status, dispatch, workstream, claimedBy, session8,
  age`), and writes an atom `now` (`{ rows, validation, counts: { ready,
  proposed, blocked }, refreshedAt, turn }`). `counts` come from one more
  `list --json` (all non-archived items) in the same refresh. A non-zero
  exit logs to debug and keeps the last good state.
- Triggers: `session.start`, `turn.complete`, and the existing
  `tool.call` handler for Bash/PowerShell when `parseAnthillCommand`
  returns a verb (any verb in `VERBS`, plus `new`, `triage`, `approve`,
  `renumber`: extend the parser's verb list for the trigger only; the
  session-link behaviour keeps its five verbs). Coalesce: at most one
  refresh in flight; a trigger during one schedules one more.

### 5.3 The pane

- Opened once per session with `$.ui.open({ id: 'anthill-now', title:
  'Anthill · <PREFIX>' })` after the first successful refresh that has at
  least one active or validation row, or immediately on `/ant`. Below the
  surface's width threshold it waits unplaced; the hook still draws when
  asked.
- `on('ui.render', { component: 'Pane', requestId: 'anthill-now' }, …)`
  draws from the `now` atom and a `selected` atom (`null` or a key):
  - **List view**: section `ACTIVE (n)` with one row per in-progress item:
    key (bold) · status pill · claimed-by · session (8 chars) · age ·
    workstream, summary on a second line; section `VALIDATION (n)` the
    same; `BLOCKED (n)` count only; footer `ready a · proposed b · blocked
    c · ↻ <age> · turn <n>` and the key hints. A row is a `Button` whose
    press sets `selected`. `placement === 'inline'` compresses each row to
    one line (key · pill · claimant · age · truncated summary).
  - **Detail view** (`selected` set): retitle the pane `Anthill · ANT-74`;
    one `$.process.run(['anthill','backlog','show', key, '--json'])` on
    selection, cached per key until the next refresh; draws title, pills
    (status, dispatch), `disposition · workstream · priority · risk`, a
    key/value block (claimed-by + session + age, reporter + created-at,
    verify, former-id when present), `SUMMARY`, `ACCEPTANCE`, the last
    three `## Log` lines, and Buttons `copy key` (`$.ui.copy`), `open file`
    (`$.prompt.fill` with `@<path>` appended so the person can ask about
    it), `transcript` (toast with the ledger transcript path), `back`
    (`role: 'dismiss'`, clears `selected`). Escape while focused clears
    `selected` first; a second Escape hands the keys back (`closeOnEscape`
    is not set: the pane stays).
- v1 draws no controller verbs (done / reject / release): the TUI Now mode
  has them. Leave a one-line comment naming the extension point.

### 5.4 Key chips in the transcript

- `on('ui.render', { component: 'AssistantMessage' }, …)` and the
  `UserMessage` twin: extract distinct keys from `e.props.text` with the
  installation's prefix (`\b<PREFIX>-[1-9][0-9]*\b`, case-sensitive). None →
  `return next(e)`. Otherwise return a tree: `{await next(e)}` first (the
  engine's row, unchanged), then a `Box` row of chips, one `Button` per
  key, label `ANT-74 · <summary or title, truncated to 40> · <status>`
  from the `now` rows when present, else `ANT-74` alone (dim) — no
  process call per row. Press → `selected = key`, open the pane if it is
  not open, `$.ui.open` with `focus: true`.
- Hover card: each chip is wrapped in a keyed `Box` holding a second `Box`
  drawn `display: "none"` with `hover: { display: "flex" }` that shows
  `key · status · dispatch · disposition · workstream`, the summary, and
  `claimed by <handle> · <age>` or `unclaimed`. Content only from cached
  state.
- `UserMessage` rows with `origin.kind` other than the composer's prompt
  (task notifications, teammate messages) are skipped.
- The hook never fails a row: wrap with `.catch(($, e, next) => next(e))`.

### 5.5 Command and status line

- `$.command.register({ name: 'ant', description: 'Open the Anthill pane
  (/ant 74 opens an item)' })` on `session.start`; `command.run`: no
  argument toggles the pane (open ↔ close); a number or key sets `selected`
  and opens; unknown key → toast.
- After each refresh: `$.ui.status('anthill <PREFIX> · <active> active ·
  <validation> validation')`, or `undefined` when both are zero.

### 5.6 Evidence for phase 3

- `claude plugin validate <repo>` clean; `tsc -p <repo>` clean once the
  types are laid; `claude plugin test <repo>` green with tests for: key
  extraction (prefix-bound, case, no `UTF-8`/`SHA-256` false positives),
  row model from fixture JSON (two active, one validation, counts), the
  Pane render in list and detail state from seeded atoms (`pane.test.tsx`
  pattern in the skill's examples), the chip render returning `next(e)`
  untouched when no key is present, `/ant 74` setting `selected`.
- `CHANGELOG.md` 0.4.0 entry; `README.md` section "Now pane and key
  chips"; `.claude-plugin/plugin.json` version 0.4.0, `types` pointing at
  the state contract.
- Manual check for the user after install: open a session in this
  installation, mention `ANT-1` in a prompt, see the chip; `/ant` opens
  the pane.

## 6. Out of scope (recorded decisions)

- Escalation ids join the key sequence in a later phase (K3).
- Pane controller verbs (K4) and prompt-side context injection on
  `prompt.submit` (K5) are deferred; the `¤74` typing shorthand with them.
- No database or index; files are the only truth (decision log
  2026-11-17).
