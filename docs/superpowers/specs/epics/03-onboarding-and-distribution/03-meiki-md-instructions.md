# Story: MEIKI.md Instructions Pack

**Epic:** Onboarding & Distribution
**Ref:** Design spec §7.1

## Summary

Author the canonical `MEIKI.md` file — the AI instructions that get pasted into user-global config files (`~/.claude/CLAUDE.md`, etc.). This is a content authoring task, not a code task.

## Scope

### File location

`instructions/MEIKI.md` in the repo, embedded into the binary via `go:embed`.

### Content structure (from spec §7.1)

Four sections, kept to ~50-100 lines total for low token cost:

1. **Lifecycle triggers**
   - Session start: run `meiki brief --json`, present conversationally if non-empty
   - Session end / "wrap up" / "done for the day": run `meiki review`
   - "Good morning" / "let's get to work": re-run `meiki brief --json`

2. **In-session capture** (one example per type)
   - achievement, achievement --closes, learning, blocker, todo, idea

3. **Mutations**
   - Todo done → `--closes` on achievement
   - Todo dropped → `meiki abandon`
   - Blocker resolved → `meiki resolve`
   - User correction → `meiki reopen` (AI should not call on its own)

4. **Stale item triage**
   - When brief output includes "Needs triage" section, ask user about each item

5. **Anti-patterns**
   - Don't capture AI reasoning, tool calls, navigation, trivial actions
   - Don't duplicate — use `meiki today` to check
   - Only capture what actually happened
   - If meiki is not on PATH, silently skip
   - Empty stdout = no-op, don't mention meiki

### Quality bar

- Concise: every line earns its token cost
- Explicit: triggers are unambiguous so AI compliance is high
- Tested: validated against representative session transcripts before shipping

## Acceptance Criteria

- [ ] File exists at `instructions/MEIKI.md`
- [ ] Contains all four sections from spec §7.1
- [ ] Under 100 lines of markdown
- [ ] Embedded in the binary and printed by `meiki setup`
- [ ] Language is imperative and unambiguous (not suggestive)
