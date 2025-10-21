# PRD: Textual-based TUI for Playground Setup

## Executive Summary
We will replace the current Rich-only progress UI with a Textual-based application backed by an event-driven task engine. The core change is to separate state from rendering: the engine emits structured, idempotent events; a coordinator reduces those events to per-task snapshots; the UI renders snapshots at a stable cadence. This eliminates flicker, duplicate lines, and timing race conditions we are currently fighting with incremental fixes.

## Why Change (Problems with Pure Rich)
The current Rich-based approach directly updates progress from task methods and logs:
- Interleaved output: Console INFO lines appear alongside progress bars (duplication and noise).
- Timing flicker: Bars “jump” or stall (e.g., sit on "Checking prerequisites" then jump to health), due to ad‑hoc throttling and lack of deterministic state.
- Duplicate messages: Retries/health messages can appear twice without a central authority for idempotency.
- Non-deterministic progress: Progress math is embedded in UI callbacks; long steps that don’t emit intermediate updates appear as jumps.
- Growing complexity: Heuristics to hide quirks (throttles, dedupe, string truncation) are spreading across UI code.

Root cause: Rendering is tightly coupled to task execution. Without a single source of truth and a proper message loop, Rich progress bars alone cannot guarantee stable, idempotent rendering under parallelism and retries.

## Solution Overview
Adopt a message-driven model with a Textual UI:
- Engine emits structured events (not strings):
  - TaskStarted, TaskCompleted, TaskFailed
  - SubtaskStarted, SubtaskCompleted
  - HealthAttemptStarted, HealthAttemptFailed, HealthCompleted
  - Each event includes: task, subtask (optional), attempt, total_attempts, timestamp, seq (monotonic), and optional payload
- Coordinator reduces events into snapshots:
  - Per-task snapshot: {state, current_subtask, completed_subtasks, health: {id, status, attempt, total}, percentage}
  - Deterministic progress: sum(completed subtask weights) + health portion; no math in the UI
  - Idempotent: last-writer-wins via monotonic seq
- Textual app renders snapshots at a fixed tick (e.g., 10 Hz):
  - No throttling/dedupe hacks in UI; renderer simply paints latest snapshots
  - Clear layout: header/footer, task list with bars, and a health panel
  - While TUI is active, suppress duplicate console INFO (keep detailed logs per-task in files only)

## Detailed Design
### 1) Event Model
- Define TaskEvent dataclass (type, task, subtask?, attempt?, total_attempts?, payload?, seq, ts)
- Events are created by BaseTask and health logic, emitted via EventEmitter (thread-safe queue)

### 2) Coordinator (Reducer)
- Single consumer of event queue
- Maintains TaskSnapshot per task:
  - state: pending | running | completed | failed
  - subtasks: list with status & weights
  - health: list of checks {id, status, attempt, total}
  - percentage: computed from subtask weights + health ratio
- Exposes read-only snapshots to the UI

### 3) Textual UI
- Components:
  - Header (title + level X/Y)
  - TaskList view: rows with name, bar, status text, elapsed
  - Health view: live status per task (current check + attempt)
  - Footer (key hints)
- Update pattern:
  - UI schedules periodic refresh (timer) and pulls snapshots from the Coordinator
  - No per-event UI updates required; the message loop handles rendering

### 4) Runner Integration
- run_all_with_tui spins up Coordinator + Textual app
- Each task execution receives an EventEmitter to report state transitions
- At task completion, engine emits a final TaskCompleted (UI drives bars to 100%)
- During TUI runs, route INFO to per-task logs; suppress duplicate console lines

### 5) Progress Semantics
- Subtasks declare weights; health portion is a configurable ratio per task (e.g., k3d = 1.0)
- Percentage = completed_subtasks_weight + health_completion * health_weight
- Task completion always emits 100%

## Scope & Deliverables
- MVP (1–2 days):
  - Event dataclasses and EventEmitter
  - Coordinator with in-memory reducer and read-only snapshot API
  - Textual app with a single screen: header, task list, basic health status
  - Wire k3d task to emit events (start/complete, health attempts)
  - Suppress duplicate console lines during TUI
- Polished (3–4 days total):
  - Better layout and keyboard controls (expand/collapse tasks)
  - Task log viewer modal (tail per-task logfile)
  - Error panel with subtask/health failure details
  - Persisted run summary (JSON) for post-mortem

## Migration Strategy
1. Introduce events + coordinator under a feature flag (keep current Rich path as fallback)
2. Switch K3dTask to emit events; verify stable progress and health visuals
3. Migrate remaining class-based tasks incrementally
4. Optionally convert legacy function-based tasks or wrap them in thin event shims

## Risks & Mitigations
- Concurrency: Events arriving out-of-order → use seq and last-writer-wins in reducer
- Terminal compatibility: Textual is widely compatible; keep Rich fallback flag (--tui=rich)
- Learning curve: Provide a small event API; hide Textual specifics behind renderer abstraction

## Success Metrics
- No duplicate lines in TUI runs (bars only; console shows summary)
- Progress is monotonic and ends at 100% for successful tasks
- Health checks show attempts and pass/fail clearly
- Reduced UI code complexity (no scattered throttles/dedupe)

## Open Questions
- How much detail to show for health (all checks vs current check)?
- Should tasks be allowed to emit intermediate “phase” events within long subtasks, or should we split them as separate subtasks?

## Appendix: Why Textual (vs staying with Rich)
- Textual provides a proper message loop, timers, diffed rendering, and layout primitives, so the UI becomes a pure function of state.
- Rich alone can render bars, but without a central coordinator and event loop, the UI requires ad‑hoc throttling/dedupe to look stable under concurrency.
