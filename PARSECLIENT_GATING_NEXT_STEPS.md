# ParseClient Gating Next Steps

This branch proves that parser-level gating in `detector.parseClient` can produce a large runtime win on the 500-sample, but it is still a prototype.

Current prototype result on the clean `optimization-round` base:
- baseline: `477/500` passed, `16.716s`
- gating prototype: `480/500` passed, `9.182s`

## What this branch does

- Adds configurable exact app-id shortcuts before the normal parser chain:
  - browser app-id -> jump straight to browser parsing
  - mobile app app-id -> return mobile app immediately
- Adds positive-token gates before these parsers when full gating is enabled:
  - `FeedReader`
  - `MobileApp`
  - `MediaPlayer`
  - `PIM`
  - `Library`
- Keeps `Browser` always running in its normal order slot.

Current modes exposed through `detector.WithClientParserGating(...)`:
- `ClientParserGatingFull` - exact shortcuts plus heuristic parser gates (current default on this branch)
- `ClientParserGatingExactOnly` - exact app-id shortcuts only
- `ClientParserGatingDisabled` - original parser chain with no gating shortcuts

## Why this needs cleanup before rollout

- The gate token lists are hand-tuned from a sample, not derived systematically.
- Some tokens are broad and could age poorly.
- Gating logic currently lives directly in `pkg/detector/detector.go`, which is good for the experiment but not ideal for long-term maintenance.
- The sample improved, but this still needs broader compatibility validation.

## Recommended next steps

1. Measure parser invocation counts directly
- Add internal counters for each client parser attempt and success.
- Record both parser calls and candidate regex attempts.
- Confirm which parsers are truly dominant on larger samples and on warm runs.

2. Split heuristics from orchestration
- Move gating rules into small helper types or files such as:
  - `pkg/detector/client_gates.go`
  - `pkg/detector/client_shortcuts.go`
- Keep `parseClient` focused on ordering and result assignment.

3. Replace ad-hoc token lists with parser-owned signals
- Derive positive gate tokens from parser metadata where possible.
- For parsers with strong exact hints, prefer exact lookup over substring lists.
- For token-gated parsers, consider exposing parser-owned `ShouldTry` logic instead of embedding knowledge in `detector`.

4. Separate high-confidence and heuristic shortcuts
- High-confidence shortcuts:
  - app-id exact matches
  - parser-specific exact identifiers
- Heuristic shortcuts:
  - UA substring gates
- Treat these as different layers so they can be measured and tuned independently.

5. Use the rollout switch to compare modes cleanly
- The branch now has `WithClientParserGating(...)`.
- Use it to compare:
  - no gating
  - exact-hint-only gating
  - full heuristic gating
- Keep future experiments measurable by mode instead of mixing behavior behind the default path.

6. Audit false negatives systematically
- Re-run sampled compatibility on multiple seeds and larger sizes.
- Group misses by parser family and by skipped parser.
- Pay special attention to long-tail mobile apps and library-style clients.

7. Revisit browser work after gating stabilizes
- Browser still runs on every request.
- Once early parsers are better controlled, inspect whether browser can skip expensive UA regex work when app-id or client hints already identify the browser strongly enough.

8. Consider parser-local indexes before more detector heuristics
- If a parser still dominates after gating, improve its candidate selection first.
- This is likely safer than continuously growing global token lists.

## Suggested immediate follow-up experiments

1. Add per-parser counters and compare 500, 2000, and warm-run measurements.
2. Extract gating helpers into dedicated files without changing behavior.
3. Add CLI flags so sampled runs can switch gating mode without code edits.
4. Compare exact-hint-only gating against full heuristic gating.
5. If mobile app remains expensive, add parser-owned fast-paths instead of more detector token growth.

## Exit criteria for a production-worthy version

- Sampled compatibility is at least as good as the clean baseline across repeated runs.
- Parser invocation counters show a stable runtime reduction, not just a lucky sample.
- Gating logic is isolated, documented, and covered by targeted tests.
- Exact-hint shortcuts and heuristic gates are independently measurable.
