# Rollout PR template — go-pkg

Used by `/open-config-pr` and the manual rollout flow. Values come straight
from this file; do not improvise.

## Branch
`ai/claude-config-setup`

## Title
`chore(ai): add Claude Code config`

## Commit message
```
chore(ai): add Claude Code config

Extends the existing .claude/ with rules, hooks, and configs so every
contributor gets the same scaffolding-aware context when working on
go-pkg's multi-module layout.

- rules/: code-style, api-stability, backward-compatibility, testing,
  error-handling, module-boundaries
- hooks/: gofmt-on-write (PostToolUse), block-env-files (PreToolUse)
- settings.json: wired hooks to the new scripts, enabled shared plugins
- CLAUDE.md: @imports the new rule files so they auto-load
- configs/pr-template.md: single source of truth for rollout PRs

No product code or package APIs changed.
```

## PR body
```markdown
## Summary
- Extends the existing `.claude/` with `rules/`, `hooks/`, and `configs/`
  aligned to the DataForce AI-config standard.
- Captures `go-pkg`'s actual conventions (multi-module layout, semver per
  module, cross-module release order) as explicit rule files.
- Moves the inline gofmt hook into a tracked script and adds a
  PreToolUse block for `.env*` / credential paths.

## What's new
- `.claude/rules/*` — code-style, API stability, backward compatibility,
  testing, error handling, module boundaries.
- `.claude/hooks/gofmt-on-write.sh` — gofmt + goimports on edited `.go`
  files. Non-blocking.
- `.claude/hooks/block-env-files.sh` — refuses Edit/Write against env
  files, pem, credentials, service-account JSON.
- `.claude/settings.json` — enabledPlugins for shared marketplace,
  PostToolUse + PreToolUse wired to the new scripts.
- `CLAUDE.md` — adds a Rule Imports section so every rule auto-loads.

## What's unchanged
- Existing `agents/code-reviewer.md`, `skills/add-package`, and
  `skills/release-package` are kept intact.
- No module, `go.mod`, or package API touched.

## Test plan
- [ ] Diff review: `.claude/` and `CLAUDE.md` only.
- [ ] `shellcheck .claude/hooks/*.sh` clean.
- [ ] Edit a `.go` file via Claude Code and confirm it's reformatted.
- [ ] Edit a dummy `.env` and confirm the hook blocks with exit 2.
```

## Reviewers
Primary: whoever CODEOWNERS lists at repo root.
