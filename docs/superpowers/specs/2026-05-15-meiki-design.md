# meiki — Design Spec

**Status:** Draft, pending user review
**Date:** 2026-05-15
**Scope:** v1 — the "memory loop" only

---

## 1. Overview

`meiki` is a local command-line tool that captures work memory automatically during AI CLI sessions, produces daily reviews, and delivers a briefing the next morning. The user does no manual journaling.

The user's premise: existing productivity apps still require administrative overhead (logging what was done, planning tomorrow, reflecting). When most work happens through AI CLI tools (Claude Code, GitHub Copilot CLI, others), the AI is already a witness to that work — it should record it so the human doesn't have to.

`meiki` is the local memory store + CLI surface that makes this work across any AI CLI tool. The contract is the CLI commands and the file format, not the model.

## 2. Goals (v1)

- **Memory loop:** capture in-session → end-of-day review → next-morning brief that carries forward unfinished items.
- **Zero manual administration** in normal use. The user types `meiki` directly only for setup or correction.
- **Model/tool-agnostic.** Works with any AI CLI that can read user-level instructions and run shell commands.
- **Local-only, no daemon, no UI.** Data lives in plain files on disk.
- **Honest history.** The append-only log preserves what happened (created, closed, reopened) so the data is suitable for future self-review.

## 3. Non-goals (v1)

Out of scope for the first version. Some return in v2+ (see §10).

- External integrations (JIRA, Outlook, Slack, calendar).
- Multi-profile / multi-context on a single machine.
- Web or desktop UI.
- Background daemon, scheduled reminders, proactive pings.
- Cross-machine data sync.
- Search, full-text index, or analytics dashboards.
- Configurable entry types or custom taxonomies.
- Sharing, multi-user, or team features.

## 4. Architecture

Three layers, strict separation. Each can be understood and replaced independently.

### 4.1 The CLI binary (`meiki`)

A single static Go binary. Sole owner of on-disk state. Exposes a small set of commands described in §6. Knows nothing about specific AI tools, MCP servers, JIRA, calendars, or any external system — those are not its concern.

All "smarts" live here: when to debounce a brief, what carries forward, how to roll up a daily review, how to format output. The AI does not make these decisions.

### 4.2 The instructions pack (`MEIKI.md`)

A canonical markdown file shipped with the binary. The user pastes its content (or `@include`s it) into the **user-global** instructions file of each AI tool they use:

- Claude Code: `~/.claude/CLAUDE.md`
- Copilot CLI: its user-level equivalent
- Generic / other tools: `AGENTS.md` or whichever convention applies

`MEIKI.md` lives at the user level, not per-repo. Every repo automatically participates without scaffolding. Repo awareness is the AI's job; `meiki` is deliberately repo-blind.

### 4.3 Optional Stop hook (Claude Code)

A one-line entry in `~/.claude/settings.json` that runs `meiki review --silent` when a Claude Code session ends. Purely a safety net: if the AI followed its instructions and called `meiki review` on its own, the hook is a no-op (debounced). If it didn't, the hook ensures the day's rollup is fresh anyway.

For AI tools without an equivalent hook, behavior is identical *if* the AI's in-instruction call to `meiki review` ran. Instructions are belt; hook is suspenders.

## 5. Data Model

JSONL append-only log + generated markdown reviews. Open-item state is derived by scanning the log, not maintained as a separate index.

### 5.1 Entry shape

One JSON object per line in the JSONL file. Common fields on every entry; type-specific fields appear only where relevant.

```json
{
  "id": "01HXY9...",
  "ts": "2026-05-15T09:14:22Z",
  "type": "todo",
  "content": "Ping Sam re: API contract — he's in CET so before 11am",
  "project": "meiki",
  "tags": ["external", "ping"],
  "source": "cli",

  "priority": "tomorrow",
  "due": "2026-05-16",
  "status": "open",
  "closes": null,
  "external_ref": null
}
```

**Common fields (always present):**

| Field | Type | Notes |
|---|---|---|
| `id` | ULID string | Addressable — the AI references entries by id for mutations. |
| `ts` | ISO-8601 timestamp (UTC) | When the entry was captured. |
| `type` | enum | `achievement` \| `learning` \| `blocker` \| `todo` \| `idea`. Closed set in v1. |
| `content` | string | One concise sentence. AI-summarized. |

**Common fields (optional):**

| Field | Type | Notes |
|---|---|---|
| `project` | string | Repo or working directory context at capture time. |
| `tags` | string[] | Free-form, no taxonomy. |
| `source` | enum | `cli` \| `hook` \| `manual`. How the entry was created. |
| `external_ref` | string | Forward-friendly: reference like `"jira:ENG-1234"`. v1 stores and returns; v2+ uses for dedupe across MCP-sourced data. |
| `supersedes` | ULID string | Present on mutation entries (abandon / resolve / reopen). References the prior state-bearing entry for the same logical item. See §5.5. |

**Type-specific fields:**

| Type | Fields | Notes |
|---|---|---|
| `todo` | `priority` (`tomorrow` \| `this-week` \| `someday`), `due` (ISO date, optional), `status` (`open` \| `done` \| `abandoned`) | Carry-forward applies while `status:open`. |
| `blocker` | `status` (`open` \| `resolved`) | Carry-forward applies while `status:open`. |
| `achievement` | `closes` (id of a todo this achievement completes, optional) | Linking `--closes` to an achievement is the preferred way to mark a todo done. |
| `learning` | — | No status. Journaled; does not carry forward. |
| `idea` | — | No status. Journaled; does not carry forward unless explicitly promoted to a todo later. |

The parser is permissive on unknown fields — older binaries reading newer files won't crash. This keeps schema evolution painless.

**Concurrent write safety:** Multiple AI sessions may call `meiki capture` simultaneously. The implementation must open JSONL files with `O_APPEND` so that writes under `PIPE_BUF` (typically 4096 bytes on POSIX) are atomic. Single-sentence JSONL entries comfortably fit within this limit. No external locking is required for v1.

### 5.2 File layout

```
~/.local/share/meiki/
├── entries/
│   └── 2026/
│       └── 05/
│           └── 2026-05-15.jsonl
├── reviews/
│   └── 2026/
│       └── 05/
│           └── 2026-05-15.md
└── state.json
```

- **`entries/YYYY/MM/YYYY-MM-DD.jsonl`** — append-only daily log. One line per entry. Full ISO date in filename so files are self-describing if moved or grepped in isolation.
- **`reviews/YYYY/MM/YYYY-MM-DD.md`** — human-readable daily review, generated by `meiki review`. Template-based (grouped entries by type: achievements, learnings, blockers encountered, ideas, open items carried forward). Meiki produces this without an LLM — the narrative quality comes from the AI presenting it conversationally during `meiki brief`, not from the review file itself. Regenerable; never authoritative.
- **`state.json`** — small JSON file tracking `{"last_brief_ts": "<ISO-8601>", "last_review_ts": "<ISO-8601>"}` for debouncing. Both are full timestamps (not dates) so `meiki brief` can detect "new entries since last brief" within the same day. Implementation detail; users do not edit.

Year/month nesting keeps directories manageable over multi-year use without adding a per-day folder that would contain a single file.

### 5.3 Carry-forward, derived not stored

When `meiki brief` (or any open-items query) runs, it scans the JSONL files in `entries/YYYY/MM/` walking back from today and computes:

- **Open todos** = entries with `type:todo` and `status:open`, *not* superseded by a later mutation (a later `type:achievement` with matching `closes:<id>`, or any later entry referencing the same id with a non-open status).
- **Open blockers** = same logic, `type:blocker`.

Mutations are always written as **new entries** that reference the prior `id`. Nothing is ever overwritten in place. This preserves an honest record for self-review.

Default scan window: 30 days back. Configurable via `[ui] open_scan_days` in `config.toml`. With one short sentence per entry, even heavy use stays well under a megabyte per year — scans are effectively free.

### 5.4 Priority decay and stale item triage

A todo with `priority: "tomorrow"` created on Monday is still labeled "tomorrow" on Wednesday unless explicitly updated. To prevent silent accumulation of stale high-priority items:

- **Days 1–3 overdue:** `meiki brief` flags the item with its age (e.g., "2 days overdue") but still includes it in the open list normally.
- **Day 3+ overdue:** `meiki brief` moves the item to a dedicated "Needs triage" section in its output. The AI's `MEIKI.md` instructions tell it to ask the user how to handle each stale item: re-prioritize (update priority to `this-week` or `someday`), set a new `due` date, or `meiki abandon` with a reason.

"Overdue" is calculated relative to the original `priority` or `due`:
- `priority: "tomorrow"` → overdue if today > capture date + 1 day.
- `priority: "this-week"` → overdue if today > end of the week the todo was captured in.
- `priority: "someday"` → never overdue (no urgency implied).
- `due: "YYYY-MM-DD"` → overdue if today > due date (explicit due takes precedence over priority-based calculation).

Re-prioritization produces a new mutation entry (same `supersedes` mechanism as other mutations) so the original priority is preserved in history.

### 5.5 Mutation entries

Closing or modifying state on a prior entry produces a new entry of one of these shapes:

- **Closing a todo with completion:** an `achievement` entry with `closes: <todo_id>`. Preferred path — captures both the closure and the accomplishment in one entry.
- **Abandoning a todo:** a synthetic entry recording `{type: "todo", status: "abandoned", id: <new>, supersedes: <old_id>, content: "<reason>"}`. Produced by `meiki abandon`.
- **Resolving a blocker:** analogous, `{type: "blocker", status: "resolved", supersedes: <old_id>, content: "<how>"}`. Produced by `meiki resolve`.
- **Reopening:** `{type: <todo|blocker>, status: "open", supersedes: <old_id>, content: "reopened"}`. Produced by `meiki reopen`.

The `supersedes` field references the most recent state-bearing entry for that logical item.

## 6. Command Surface

Five small groups. Most are AI-called; a few are user-facing for setup and override.

### 6.1 Lifecycle (AI-called)

| Command | When | Behavior |
|---|---|---|
| `meiki brief [--json]` | Session start | Outputs the morning briefing: most recent review summary + open todos (ranked by priority, stale items in a "Needs triage" section) + open blockers. **Debounced** — returns empty if already produced today *and* no new entries since last brief timestamp. Empty output = no-op; AI proceeds. On a fresh install with no history, outputs a one-line welcome message. |
| `meiki review [--silent]` | Session end (Stop hook), or "let's wrap up" | Incremental: regenerates today's review markdown from today's entries. Safe to call repeatedly. `--silent` suppresses stdout (for hook use); review is still written to disk. |
| `meiki capture <type> "<content>" [options]` | Throughout the session | Appends an entry. Prints the new entry's `id` on stdout. Options: `--project <p>`, `--tags a,b`, `--priority tomorrow|this-week|someday`, `--due YYYY-MM-DD`, `--closes <id>`, `--external-ref <ref>`. |

### 6.2 Mutation (AI-called)

| Command | Purpose |
|---|---|
| `meiki capture --type achievement --closes <id> "..."` | Preferred path for completing a todo. One entry; both effects. |
| `meiki abandon <id> ["reason"]` | Mark a todo `abandoned`. Different from `done` for self-review accuracy. |
| `meiki resolve <id> ["how"]` | Mark a blocker `resolved`. |
| `meiki reopen <id>` | Undo a close/abandon/resolve. For "AI got it wrong, this is still on my plate." |

No standalone `meiki close` — the verb is split into `--closes` (with achievement) and `abandon` (without). The split forces the AI to be honest about *why* a todo went away.

### 6.3 Query (read-only)

| Command | Output |
|---|---|
| `meiki open [--json]` | Open todos + blockers, grouped. Used by `meiki brief` internally; also handy for mid-day "what's on my plate?" |
| `meiki today [--json]` | Today's entries grouped by type. For mid-session "what have I logged so far?" |
| `meiki recent [--days N] [--type T] [--json]` | Filtered tail. Defaults: last 7 days, all types. |

### 6.4 Setup (user-facing)

| Command | Purpose |
|---|---|
| `meiki setup` | One-time per machine. Creates the data dir, prints the `MEIKI.md` content and Stop-hook JSON snippet for the user to paste into their global AI configs. Idempotent — re-run any time to re-print. |
| `meiki doctor` | Self-diagnostic: data dir exists and is writable, `state.json` parses, hook command is on PATH. Reports what it finds. |

### 6.5 Output conventions

- **Default = human-readable text.** `meiki brief` outputs prose; `meiki open` outputs a grouped list with ids. The CLI is useful standalone.
- **`--json` available everywhere it makes sense.** Same data, structured. AI tools use it via the instructions in `MEIKI.md`.
- **Capture commands print the entry id on stdout, diagnostics on stderr.** Lets the AI script `id=$(meiki capture ...)`.
- **Empty stdout = no-op.** Debounced calls exit 0 with zero bytes. AI instructions treat empty as "nothing to surface, move on."
- **Exit codes:** 0 success, 1 user error (bad args, unknown id), 2 internal error.

### 6.6 Deliberately excluded from v1

- Search, full-text grep, or filter-by-content commands. JSONL is greppable; revisit when a real workflow demands it.
- Edit or delete of past entries. Append-only is the contract; corrections are new entries.
- Stats, analytics, or dashboards. The daily review markdown is the dashboard.
- Sync, export, or cloud commands. Data is local-only.

## 7. AI Integration

Two artifacts: the canonical `MEIKI.md` instructions pack and the optional Stop hook script. Everything else is `meiki` doing its job.

### 7.1 `MEIKI.md` structure

Concise enough to keep token cost low on every session, explicit enough that compliance is high. Four sections:

**Lifecycle triggers:**
- At session start (before responding to the user): run `meiki brief --json`. If non-empty, present conversationally. If empty, proceed.
- At session end, or on phrases like "wrap up" / "done for the day" / "let's review": run `meiki review`. Do not surface its output unless the user asked explicitly — the review feeds tomorrow's brief.
- On phrases like "good morning" / "let's get to work": re-run `meiki brief --json` (self-debounces).

**In-session capture (with one example per type):**
- `meiki capture --type achievement "..."` — for completed work.
- `meiki capture --type achievement --closes <id> "..."` — to complete a previously-tracked todo.
- `meiki capture --type learning "..."` — for non-obvious things learned.
- `meiki capture --type blocker "..."` — for blockers encountered.
- `meiki capture --type todo --priority <p> "..."` — for new todos.
- `meiki capture --type idea "..."` — for half-formed thoughts worth remembering.

**Mutations:**
- Todo done → prefer `--closes <id>` on an achievement.
- Todo dropped → `meiki abandon <id> "reason"`.
- Blocker resolved → `meiki resolve <id> "how"`.
- User correcting AI → `meiki reopen <id>`. AI should not call `reopen` on its own; only when the user explicitly corrects.

**Stale item triage:**
- When `meiki brief --json` output includes a "Needs triage" section (items 3+ days overdue), ask the user about each one before starting other work: re-prioritize, set a new due date, or abandon with a reason. Use the appropriate mutation command for each decision.

**Anti-patterns (the part that prevents noise):**
- Do not capture AI reasoning steps, tool calls, or navigation.
- Do not capture trivial actions (single file reads, simple edits).
- Do not duplicate. If unsure, `meiki today` shows what's already logged.
- Only capture what actually happened. No speculation, no fabrication.
- If `meiki` is not on PATH, silently skip captures and continue. Optionally surface to the user once per session.
- Empty stdout from any `meiki` command = no-op. Do not mention meiki to the user when output is empty.

### 7.2 Stop hook (Claude Code)

Registered in `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [{
      "matcher": "*",
      "hooks": [{"type": "command", "command": "meiki review --silent"}]
    }]
  }
}
```

Fires on every session end; the command writes/updates today's review markdown silently. Safe to fire repeatedly (incremental). Where hooks are unavailable, the AI's `MEIKI.md` instructions still call `meiki review` at session end — same behavior, marginally less reliable.

### 7.3 End-of-day flow in practice

The user's original mental model was "at end of day, AI gives me a review." Actual flow:

- Each session ends → Stop hook updates today's review markdown silently.
- Next morning, first session → `meiki brief` presents yesterday's review summary as part of the brief.

The review is *delivered* at the moment it's most useful: the start of the next workday, when the user is open to context, not when they are shutting down. The user can still request a mid-day review explicitly ("let's review the day now") which runs `meiki review` non-silently and surfaces the output immediately.

### 7.4 Graceful degradation

- **AI forgets mid-session captures:** Stop hook still produces a review of whatever *was* captured. Sparse > wrong.
- **Tool has no hook support:** behavior identical *if* in-instruction `meiki review` runs.
- **AI captures something incorrectly:** never silently destructive. Corrections via `meiki reopen <id>` are new entries; history is intact.
- **`meiki` not installed:** instructions say to skip capture and proceed; optionally surface once.

## 8. Configuration

Config file: `~/.config/meiki/config.toml`. Optional — meiki ships sensible defaults and runs without it.

**v1 keys:**

```toml
[ui]
# Max open todos shown in brief before truncating. Default: 20.
brief_max_open_todos = 20
# Days to scan back for open items. Default: 30.
open_scan_days = 30
# Days before a stale priority triggers the "Needs triage" section. Default: 3.
stale_triage_days = 3
```

Meiki uses XDG-style paths on all platforms (`~/.config/meiki/` for config, `~/.local/share/meiki/` for data) for consistency and dotfile-friendliness, even on macOS where the native convention is `~/Library/`. This is common practice for developer CLI tools and keeps the sync story simple.

## 9. Distribution & Install


- **Language:** Go. Single static binary, no runtime dependencies, easy cross-platform builds.
- **Install paths:**
  - Primary: `brew install meiki` (Homebrew tap, once published).
  - Secondary: `go install github.com/<owner>/meiki/cmd/meiki@latest`.
  - Tertiary: prebuilt binaries on GitHub releases.
- **First-run flow:**
  1. User runs `meiki setup`.
  2. `meiki setup` creates `~/.local/share/meiki/` and prints:
     - The `MEIKI.md` content the user pastes/`@include`s into `~/.claude/CLAUDE.md` and any other AI tool user-level instructions file.
     - The JSON snippet to paste into `~/.claude/settings.json` for the Stop hook.
  3. User completes the paste step. `meiki doctor` verifies.
- **Repo layout (target):**
  ```
  meiki/
  ├── cmd/meiki/                 # main package
  ├── internal/                  # core logic (entries, scan, render)
  ├── instructions/MEIKI.md      # canonical instructions pack
  ├── docs/                      # README, install, this spec
  └── ...
  ```

## 10. Out of Scope (v2+ and possibly never)

### Deferred to v2+ (natural next steps once the memory loop earns its keep)

- **Non-CLI capture surfaces.** Friendlier paths for logging meetings, side thoughts, etc. — menubar shortcut, hotkey, voice-to-text wrapper that produces `meiki capture` calls. `meiki capture` from any shell already covers the technical case.
- **MCP-augmented context in briefings.** Add a paragraph to `MEIKI.md`: "If you have MCP servers connected (JIRA, Outlook, Slack), opportunistically check them as part of the brief and capture significant items." Meiki itself remains integration-blind, credential-free, and authentication-free. The AI's own MCP connections do the work; meiki stores whatever the AI captures.
- **Dedup of MCP-sourced data** via the already-reserved `external_ref` field. AI checks `meiki open --json` before re-capturing an item with an existing `external_ref`.
- **Scheduled proactive reminders.** Cron-driven `meiki tick` that surfaces time-sensitive items. Additive.
- **Self-review export.** `meiki rollup --since <date> --kind quarterly` synthesizes a self-review draft (themed achievements, recurring blockers, ideas considered vs acted on). The long-game payoff of preserving honest history.
- **Profiles.** If multi-context-on-one-machine ever becomes a real need, wrap data dir resolution with a profile name. Purely additive.

### Possibly never

- Web or desktop UI (defeats the "not another productivity app" goal).
- Background daemon for capture.
- Multi-user, team, or sharing features.
- Cross-machine data sync.
- Full-text search or analytics dashboards.

### Tempting but rejected for v1

- Tag taxonomies or autocomplete.
- A `meeting` entry type (folded into `achievement` with `tags:["meeting"]` until non-CLI capture matures).
- Configurable entry types.
- Schema validation beyond required fields.

## 11. Open Questions & Risks

- **AI compliance with capture instructions.** The biggest risk. Mitigations: tight `MEIKI.md` with explicit triggers; Stop hook as a floor; `meiki today` for self-check. Will need iteration after real-world use; expect to revise `MEIKI.md` based on observed AI behavior.
- **Token cost of `MEIKI.md` loaded on every session.** Keep the file lean (~50–100 lines of markdown). Measure after v1 ships; if it's a real problem, consider on-demand loading patterns specific to each AI tool.
- **Auto-close correctness.** AI may close the wrong todo or miss a closure. `meiki reopen` is the safety valve; corrections are non-destructive (new entries, not edits).
- **Brief noise vs. signal.** Over time the carry-forward could accumulate stale todos. Mitigations: AI is instructed to abandon-with-reason rather than ignore, and `meiki brief` can prioritize/limit (`brief_max_open_todos` in config).
- **Idea persistence.** Ideas don't carry forward in the daily brief in v1. Open question: how does the user surface them later? Likely a v2+ `meiki ideas` command, or a dedicated section in the weekly rollup when that lands.
- **`source` semantics.** `cli` | `hook` | `manual` is the planned enum, but the distinction between `cli` (AI called from within a session) and `manual` (user ran `meiki capture` directly) may be hard to detect reliably. Acceptable to start with all captures defaulting to `cli` and revisit.

## 12. Acceptance Criteria for v1

The v1 implementation is complete when:

- [ ] `meiki capture` writes a well-formed JSONL entry to the correct daily file.
- [ ] `meiki brief` produces a human-readable briefing including open todos and yesterday's review summary, debounces on same-day repeat calls, and emits empty output when there is genuinely nothing to say.
- [ ] `meiki review` produces today's review markdown idempotently; safe to run repeatedly during the day.
- [ ] `meiki capture --closes <id>` correctly marks the referenced todo as completed (verified by `meiki open` no longer listing it).
- [ ] `meiki abandon`, `meiki resolve`, `meiki reopen` produce the documented mutation entries and update derived open state.
- [ ] `meiki open`, `meiki today`, `meiki recent` produce correct human-readable and `--json` output.
- [ ] `meiki brief` flags items 1–3 days overdue with their age and moves 3+ day overdue items to a "Needs triage" section.
- [ ] `meiki setup` prints actionable snippets; `meiki doctor` produces a useful self-check.
- [ ] The Stop hook runs `meiki review --silent` without errors and updates the day's review markdown.
- [ ] `MEIKI.md` content is concrete enough that an AI tool following it captures appropriate entries during a representative session.
- [ ] The binary is distributable as a single static artifact for macOS (intel + arm) and Linux at minimum.
