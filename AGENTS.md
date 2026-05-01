# AGENTS

## 1) Non-Negotiable Workflow Rule

Before claiming any TUI fix is done:

1. Start the app locally (`go run .`).
2. Manually walk the full wizard flow end-to-end.
3. Verify rendering, navigation, focus, and key behavior in every step.
4. Only then report completion.

If full verification was not done, state that explicitly and do not claim completion.

## 2) TUI Architecture Rules

1. Prefer centralized layout logic and shared pane rendering helpers.
2. Avoid per-screen hacks and duplicated rendering logic.
3. Keep styling/spacing behavior consistent across all steps.
4. Fix root causes, not cosmetic one-off patches.

## 3) Wizard Layout Contract

### Global

1. Wizard uses consistent pane behavior and border styles.
2. Active pane is indicated by top-border highlight.
3. Panes must never overflow terminal height.
4. Long lines must be truncated with `…` (no wrapping that breaks layout).
5. Scroll must work when content exceeds visible area.

### Intro Line

1. Every wizard step must show a top intro/description line.
2. Intro line is first visible row of the screen.
3. Keep exactly two blank lines between intro line and pane block.

### Confirm Pattern

1. Use one unified confirm label: `> Confirm [Ctrl+S]`.
2. Do not show a separate `Ctrl+S` line below confirm.
3. Inside a pane:
   - step headline
   - 1 blank line
   - `> Confirm [Ctrl+S]`
   - 2 blank lines
   - remaining content

## 4) Step-Specific Contract

### Step 1 (Source Directory)
1. Single full-width pane.
2. Source browser, selected source path, source stats, and source event analysis are in the same pane.
3. Show `> Confirm [Ctrl+S]` as per Confirm Pattern.

### Step 2 (Target Directory)

1. Two panes:
   - left: selected source summary
   - right: target selection + create preview + browser
2. Filepicker rows must never wrap; truncate with `…`.

### Step 3+ (Preflight/Execute/Verification)

1. Keep the same visual structure and spacing rules.
2. Right side must be a single pane (no nested/double pane borders).
3. Left pane shows cumulative confirmed context.

## 5) Keybinding Contract

1. `Ctrl+D` is the only hard cancel/quit key.
2. `Ctrl+C` does not quit.
3. `q` does not quit.
4. `Esc` does not quit.
5. `Ctrl+S` confirms in every step.
6. `b` goes to previous step where applicable.
7. `Tab` switches pane focus only when multiple panes exist.

## 6) Navigation and Focus Rules

1. Focus and scrolling must be deterministic and visible.
2. If a step has one pane, focus switching must not break navigation.
3. Arrow keys must control the expected focused component.
4. Parent entry (`..`) behavior must be explicit and consistent.

## 7) Regression Prevention

1. After each UI change:
   - run `go fmt`
   - run `go test ./...`
   - run manual TUI walkthrough (`go run .`)
2. Check Step 1, Step 2, Step 3, Step 4 explicitly.
3. Verify:
   - intro line visible
   - confirm label format
   - spacing pattern
   - focus highlight
   - scroll behavior
   - no overflow/clipping/wrapping regressions

## 8) Safety During Manual TUI Tests

1. Do not start real copy/delete operations during UI-only validation.
2. Stop before destructive execution unless the user explicitly requests full execute testing.

## 9) Communication Rule

1. Be precise and factual.
2. If something was not fully tested, say so directly.
3. Do not claim “fixed” before full end-to-end manual verification.
