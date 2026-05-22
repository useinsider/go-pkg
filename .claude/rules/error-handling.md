# Error handling — go-pkg

## Return, don't panic

- Library code returns errors. Never `panic` on bad input from the caller;
  that crashes their service, not ours.
- `panic` is acceptable only for truly impossible states during init
  (e.g. malformed package-internal constants). Document it as such.

## Wrap with context

- `fmt.Errorf("reading config: %w", err)` preserves the chain for
  `errors.Is`/`errors.As`. Plain `fmt.Errorf("%v", err)` breaks it.
- The `pkg/errors` package is pinned across modules (v0.9.1); prefer
  stdlib `%w` for new code unless an existing file already uses
  `errors.Wrap`.

## Sentinel errors

- Export a sentinel (`var ErrFoo = errors.New("ins<pkg>: foo")`) when
  callers need to branch on the cause. Include the package name in the
  message so logs are greppable.
- Don't export sentinels unless callers genuinely need them; each one is a
  public API commitment.

## Error codes

- `inscodeerr` carries HTTP-aware error codes. Return a `CodeErr` from a
  package only when the caller is an HTTP handler; otherwise return a
  plain error and let the caller wrap it.
- Don't import `inscodeerr` in low-level packages (cache, logger, SQS);
  they have no business knowing about HTTP.

## Logging vs. returning

- A function either returns an error or logs it. Not both — double-logging
  makes alerts noisy and hides the real source.
- The caller closest to the user (HTTP handler, SQS consumer) decides
  whether to log. Inner packages return.

## Nil error, non-nil return

- Never return `(nil, nil)` for a "lookup miss" — return a sentinel like
  `ErrNotFound`. Callers forget to check the value when the error is nil.
