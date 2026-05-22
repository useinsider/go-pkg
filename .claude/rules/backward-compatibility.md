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
  wrap them in a way that breaks `errors.Is`.
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
- Renaming a parameter (fine for callers, not fine for code that uses
  named args in generated mocks).
- Changing the order of arguments in a variadic.
- Returning a `*Result` where it used to return `Result`.

If a change feels too small to think about, think about it anyway.
