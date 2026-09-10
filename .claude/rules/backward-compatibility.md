# Backward compatibility

Rules that stop accidental breakage between the "I just added an option"
thought and the "downstream service's CI is red" consequence.

## Config struct evolution

- New fields go at the end of `Config`. Callers using positional struct
  literals — rare but legal — break on reorder.
- Every new field must behave identically to the previous version when
  left at its zero value. If that's not possible, the field belongs in a
  new constructor, not the existing `Config`.
- Never change a field's type. Add a new field, deprecate the old one.

## Interface evolution

- Exported `Interface` types are frozen between majors. A caller that
  implements the interface themselves (for testing, for a custom backend)
  will break the moment we add a method.
- Need a new capability? Define a new narrow interface and type-assert for
  it inside the package. Example: `if x, ok := client.(Flusher); ok { ... }`.

## Defaults

- Don't change a default timeout, retry count, buffer size, or logging
  verbosity in a minor version. Callers calibrate to observed behaviour
  and silently regress when it shifts.
- If a default genuinely needs to change, add an explicit knob, keep the
  old default, and document the new recommended value.

## Error types

- Exported sentinel errors (`ErrFoo`) are API. Don't rename them, don't
  wrap them in a way that breaks `errors.Is`. Their message text is API too:
  consumers alert on log lines, so leave the string alone in a lint pass.
- When introducing a richer error, wrap rather than replace: `return
  fmt.Errorf("%w: ...", ErrFoo)` keeps existing `errors.Is(err, ErrFoo)`
  checks green.

## Cross-module coupling

- `inssqs` uses `insdash` and `inslogger`. `insssm` uses `inscacheable`.
  When releasing a new version of a dependency, ensure dependents still
  compile against it *before* tagging. `scripts/check-deps.sh` pins the
  expected versions across all modules — update it in the same PR.

## The "just one line" trap

The most expensive changes in this repo don't look dangerous:
- Swapping `int` for `int64` on a config field.
- Renaming a parameter (harmless to callers and to committed mocks; only a
  regenerated mock picks up the new name — still say why in the PR).
- Changing the order of arguments in a variadic.
- Returning a `*Result` where it used to return `Result`.

If a change feels too small to think about, think about it anyway.

## What the linter is allowed to change

`.golangci.yaml` deliberately does not enforce revive's `var-naming` on
exported identifiers (`Id`, `Url`, `Sql` stay as they are) and does not ask
for `inscodeerr.CodeErr` to become `CodeError`: both would be major-version
breaks for every consumer. Unexported names, parameter names and internals
are fair game — for example `insredis.RedisInterface` parameters `min, max`
became `minVal, maxVal` / `minSlot, maxSlot` to stop shadowing the Go 1.21
builtins, which changes no signature. If a future linter finding can only be
fixed by touching an exported symbol, exclude it by path with a comment in
`linters.exclusions.rules` and record it for the next major.
