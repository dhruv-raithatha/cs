# Feature Specification: Interrupt-Driven Session Notifications

**Feature Branch**: `004-session-interrupt-notify`
**Created**: 2026-05-11
**Status**: Draft
**Input**: User description: "currently, claude sessions lets you multi-task but doesn't help you identify
where claude needs a human in the loop. A prompt/session that has been stalled for a couple of mins or
where claude asks questions should be brought front and center to a developer's attention. This way, a
developer isn't glued to the screen but instead can truly multi-task and operate in an interrupt driven
model. There is a initial script I had used available in /Users/dhruv/dev/dotfiles/claude/scripts/notify.sh
that used to work but I think we can step up our game. Let us say we recommend claude-sessions be used by
ghostty and with this assumption, how could we do this differently so that the notifications feel more
native and rich in context. Lastly, our cs setup should also offer setting up the notify scripts while we
are at it."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ghostty-Native Notification Appearance (Priority: P1)

A developer receives a macOS notification that looks and feels as if Ghostty itself generated it — correct
icon, correct app name, native macOS notification fidelity — rather than a generic alert from a third-party
notification tool. The notification carries enough context (session name, project, message preview) to be
actionable at a glance.

**Why this priority**: The visual and brand authenticity of the notification is what makes this feel like a
first-class feature rather than a hack. A notification from an unknown sender or with the wrong icon breaks
trust and gets ignored. When the notification looks exactly like Ghostty's own alerts, it is immediately
legible as "something in my terminal needs attention." This is the tight Ghostty integration that elevates
`cs` from a convenience tool to a polished workflow.

**Independent Test**: Trigger a human-in-the-loop event in a `cs` session. Verify the resulting macOS
notification: (a) displays the Ghostty app icon, (b) identifies Ghostty as the sender, (c) shows session
name, project directory, and Claude message preview, (d) is visually indistinguishable from a notification
Ghostty itself would generate natively.

**Acceptance Scenarios**:

1. **Given** a Claude session emits a human-attention event, **When** the notification appears, **Then** it
   visually matches Ghostty's native notification appearance: correct icon, correct app identity, standard
   macOS notification layout.
2. **Given** the notification appears, **When** the developer clicks it, **Then** Ghostty comes to the
   foreground and the focus lands on the tmux pane that contains the waiting Claude session.
3. **Given** the notification contains Claude's message, **When** the developer reads it, **Then** they can
   determine the nature of the request (question, permission, task complete) without opening the terminal.
4. **Given** a prior notification exists for a session, **When** a new event fires for the same session,
   **Then** the new notification replaces the old one rather than accumulating in notification center.

---

### User Story 2 - Receive Alert When Claude Needs Input (Priority: P1)

A developer running multiple Claude sessions in the background is alerted within seconds whenever any
session pauses waiting for a response or permission approval, so they can operate fully in an
interrupt-driven mode without watching any screen.

**Why this priority**: Core functional value. Without timely alerts the feature does not enable multi-tasking.

**Independent Test**: Run a Claude session via `cs` that reaches a human-in-the-loop moment. Verify a
notification fires within 5 seconds with session name, project path, and message preview. Verify clicking
it lands in the correct pane.

**Acceptance Scenarios**:

1. **Given** a Claude session is waiting for developer input, **When** the human-in-the-loop event fires,
   **Then** a notification appears within 5 seconds showing session name, project directory (last two path
   segments), and a truncated preview of Claude's message.
2. **Given** the developer is working in a different application, **When** the notification fires, **Then**
   they can switch to the relevant session in two interactions or fewer: dismiss/click notification → land
   in correct pane.
3. **Given** multiple sessions emit events concurrently, **When** all notifications appear, **Then** each
   is distinct and independently actionable — no sessions are silently coalesced.

---

### User Story 3 - Stall Detection for Idle Sessions (Priority: P2)

A developer who stepped away gets notified when a Claude session has produced no output for longer than a
configurable threshold, prompting them to check whether direction or input is needed.

**Why this priority**: Complements P1 by catching silent stops — cases where Claude finishes a task or gets
stuck without explicitly asking anything.

**Independent Test**: Start a `cs` session and leave it idle past the stall threshold (default 3 minutes).
Verify a stall notification fires with session name and idle duration. Verify no second stall notification
fires until the session becomes active and goes idle again.

**Acceptance Scenarios**:

1. **Given** a Claude session has produced no output for longer than the stall threshold, **When** the
   threshold is crossed, **Then** a notification is sent indicating the session name and how long it has
   been idle.
2. **Given** a stall notification has been sent for a session, **When** the session becomes active again,
   **Then** no further stall notifications fire until the next idle period begins.
3. **Given** a developer configures a custom stall threshold (e.g., via `CS_STALL_THRESHOLD`), **When**
   sessions are monitored, **Then** the custom threshold is respected instead of the default 3 minutes.

---

### User Story 4 - Configure Notifications and Ghostty via cs setup (Priority: P3)

A developer running `cs setup` is guided through recommending Ghostty as the host terminal (with install
instructions if absent), then offered the option to install and configure the notification system — all in
one setup flow with no manual file editing.

**Why this priority**: Discoverability and onboarding. A new developer should go from zero to receiving
Ghostty-native notifications in a single `cs setup` run.

**Independent Test**: Run `cs setup` on a clean environment without Ghostty installed. Verify: (a) Ghostty
is recommended with install instructions displayed, (b) once Ghostty is present, the notification opt-in
step is available, (c) opting in installs the notify script, configures hooks, and fires a test
notification that looks native to Ghostty.

**Acceptance Scenarios**:

1. **Given** a developer runs `cs setup` and Ghostty is not installed, **When** the terminal recommendation
   step runs, **Then** `cs setup` explains that Ghostty is the recommended terminal for the best
   notification experience and displays the install command or URL.
2. **Given** a developer runs `cs setup` and Ghostty is already installed, **When** the terminal check
   runs, **Then** `cs setup` confirms Ghostty is detected and proceeds to the notification setup step.
3. **Given** a developer opts into notification setup, **When** setup completes, **Then** the notify
   script is installed, Claude Code hooks are registered in `~/.claude/settings.json`, and a live test
   notification fires — visually confirming the Ghostty integration is working.
4. **Given** a developer opts out of notification setup, **When** setup completes, **Then** no
   notification files are installed, and re-running `cs setup` later offers the same opt-in.
5. **Given** the notification system is already installed, **When** the developer re-runs `cs setup`,
   **Then** they see the current configuration and are offered options to test, update, or remove it.

---

### Edge Cases

- What if the tmux pane for a session cannot be resolved? Notification is sent without click-to-jump
  navigation rather than suppressed entirely.
- What if Ghostty is not running at notification time? The notification still appears via the system; the
  click action may not navigate to a pane, but the alert is not lost.
- What if multiple sessions stall simultaneously? Each fires its own notification grouped by session; they
  do not collapse into each other.
- What if a session is closed while a stall timer is pending? The stall watcher MUST NOT fire for sessions
  that are no longer active in the `cs` socket.
- What if `~/.claude/settings.json` already has conflicting hook entries? Setup detects and merges rather
  than overwriting.
- What if the developer is on a non-Ghostty terminal and runs `cs setup`? The recommendation is shown, the
  install path is offered, but setup continues and falls back to a degraded notification mode (no
  Ghostty-native appearance).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST fire a notification within 5 seconds when a Claude session emits a
  human-attention event (question asked, tool permission needed, task completed).
- **FR-002**: Notifications MUST be visually native to Ghostty — the appearance, icon, and sender identity
  MUST match what Ghostty itself generates, with no visual indication of a third-party notification tool.
- **FR-003**: Each notification MUST include: the session name, the project directory (last two path
  segments), and a preview of Claude's most recent message (truncated to 120 characters if needed).
- **FR-004**: Clicking a notification MUST bring Ghostty to the foreground and orient the developer to the
  tmux pane containing the waiting session.
- **FR-005**: Notifications for the same session MUST replace prior notifications rather than stacking in
  notification center.
- **FR-006**: The system MUST detect sessions that have produced no output for longer than a configurable
  stall threshold, defaulting to 3 minutes.
- **FR-007**: `cs setup` MUST include a terminal recommendation step that identifies whether Ghostty is
  installed, explains its advantages for notification UX, and provides install instructions if it is
  absent.
- **FR-008**: `cs setup` MUST include an opt-in notification installation step that covers: notify script
  placement, Claude Code hook registration in `~/.claude/settings.json`, and a live test notification.
- **FR-009**: Stall detection MUST operate without a persistent privileged daemon, using mechanisms
  already available in tmux or the `cs` toolchain.
- **FR-010**: The notify script MUST extend the existing Claude Code hook approach so that hooks function
  whether or not `cs` is the session manager.
- **FR-011**: Developers MUST be able to fully remove the notification system via `cs setup`, leaving no
  residual hooks or scripts.

### Key Entities

- **Session**: A `cs`-managed tmux pane running Claude Code, identified by name and current working
  directory.
- **Notification Event**: A typed signal (human-in-loop, stall, task-complete) that triggers a developer
  alert and carries enough session context to be immediately actionable.
- **Stall Watcher**: A lightweight component that tracks per-session activity timestamps and emits stall
  events when the silence threshold is exceeded.
- **Notify Script**: The shell script invoked by Claude Code hooks that composes and dispatches
  notifications with full Ghostty visual fidelity, resolving the correct tmux pane for click-to-navigate.
- **Hook Configuration**: The entries in `~/.claude/settings.json` that wire Claude Code lifecycle events
  to the notify script.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer managing 4+ simultaneous Claude sessions can respond to any human-in-the-loop
  moment within 30 seconds without watching any screen.
- **SC-002**: Notification delivery from Claude event to macOS notification center occurs in under 5
  seconds, end to end.
- **SC-003**: Zero duplicate notifications accumulate in notification center for a single session event
  (replace-not-stack verified).
- **SC-004**: A developer who sees the notification can identify the session, project, and nature of the
  request without opening the terminal — assessed by user review of notification content.
- **SC-005**: Notifications are visually indistinguishable from Ghostty's own native alerts, confirmed by
  side-by-side comparison.
- **SC-006**: Full notification setup via `cs setup` completes in under 2 minutes including the Ghostty
  install check and test notification.
- **SC-007**: Stall detection adds no perceptible latency to Claude session startup or active response
  throughput.
- **SC-008**: Clicking a notification navigates to the correct session in under 2 seconds on a standard
  developer machine.

## Assumptions

- Ghostty is the recommended and primary host terminal; `cs setup` will surface install guidance for
  developers who do not yet have it.
- Ghostty's macOS notification identity (icon, sender) is used as the visual target; the implementation
  will use whichever mechanism achieves this fidelity (escape sequences, app identity spoofing, or
  Ghostty's own notification API).
- `terminal-notifier` is treated as a fallback for non-Ghostty environments and is NOT the primary
  delivery mechanism for the Ghostty-first path.
- The existing `notify.sh` in dotfiles is a reference implementation; the new notify script supersedes it.
- Stall detection uses tmux's built-in `monitor-silence` window option and `alert-silence` hook — no
  additional process is needed.
- The notify script is installed per-user (not per-project) since it is a developer tooling concern, not
  a repository concern.
- macOS is the only supported platform for v1; Linux/Windows is out of scope.
- `cs`-managed sessions are the primary target for stall detection; sessions started directly with
  `claude` outside of `cs` receive event-based notifications (via hooks) but not stall detection.
