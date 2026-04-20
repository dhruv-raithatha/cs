# Feature Specification: cs — Claude Session Manager

**Feature Branch**: `001-cs-tmux-session-manager`  
**Created**: 2026-04-20  
**Status**: Draft  
**Input**: User description: "Build cs binary that can be used on a macos shell to manage multiple claude sessions. It does so by integrating `claude` process with tmux and takes away the learning curve of getting used to tmux."

## Clarifications

### Session 2026-04-20

- Q: How does `cs` distinguish sessions it created from manually created tmux sessions? → A: All cs-managed sessions use a dedicated tmux socket directory (e.g., `~/.cs/sockets/`) so they are completely isolated from the user's regular tmux environment.
- Q: Should session names default to the working directory basename when none is provided? → A: No — session name is mandatory. The user must either select an existing session or explicitly name a new one; there is no default derivation.
- Q: How are missing dependencies handled? → A: A dedicated `cs setup` subcommand is the first step; it checks for required dependencies and optionally offers to install them. The main `cs` command assumes setup has been run.
- Q: What should `cs` do if the user is already inside a tmux session? → A: Refuse to run and print a clear message instructing the user to detach first (e.g., "Detach from your current tmux session before using cs").
- Q: How should the picker display a cs session whose Claude process has exited? → A: Show it with a visual indicator (e.g., `[dead]` tag) so the user can choose to attach, restart, or delete — no automatic cleanup.

## User Scenarios & Testing *(mandatory)*

### User Story 0 — Initial Setup (Priority: P0)

A developer installs `cs` for the first time and runs `cs setup`. The command checks for all required dependencies, reports their status, and offers to install any that are missing (e.g., via Homebrew). Once setup passes, `cs` is ready to use.

**Why this priority**: No other workflow is possible without dependencies in place. Setup is the entry point for all new users.

**Independent Test**: Run `cs setup` on a clean machine, verify it detects missing deps, offer install, and confirm a passing status once they are present.

**Acceptance Scenarios**:

1. **Given** a required dependency is missing, **When** the user runs `cs setup`, **Then** the tool reports which dependency is absent and offers to install it.
2. **Given** the user accepts the install offer, **When** installation completes, **Then** `cs setup` re-checks and confirms the dependency is now satisfied.
3. **Given** all dependencies are present, **When** the user runs `cs setup`, **Then** the tool reports all checks passed and the user can proceed to use `cs`.

---

### User Story 1 — Launch or Resume a Session (Priority: P1)

A developer runs `cs` from any directory. The default action is to create a new session — the picker opens with `[ + new session ]` pre-selected at the top so a single Enter creates a fresh session. Existing sessions are listed below for the user to switch to if needed. No tmux knowledge is required.

**Why this priority**: This is the entire core value — frictionless creation and resumption of Claude sessions without knowing tmux commands. New session is the most common action and should require the least friction.

**Independent Test**: Run `cs` with zero existing sessions, create a named one, verify it attaches to a running Claude process. Then detach and run `cs` again — verify new session is pre-selected and the prior session is listed below it.

**Acceptance Scenarios**:

1. **Given** no existing cs sessions, **When** the user runs `cs`, **Then** the tool skips the picker and prompts directly for a new session name.
2. **Given** one or more existing cs sessions, **When** the user runs `cs`, **Then** a fuzzy-searchable picker opens with `[ + new session ]` as the first (pre-selected) entry, followed by existing sessions listed by name and working directory.
3. **Given** the user presses Enter without navigating away from the default, **When** the picker confirms, **Then** the new-session flow begins and prompts for a name.
4. **Given** the user navigates to an existing session in the picker and selects it, **When** they confirm, **Then** they are attached to that tmux session exactly where they left off.
5. **Given** the user chooses to create a new session and provides a name, **When** confirmed, **Then** a new tmux session is created in the cs socket directory and Claude starts inside it.

---

### User Story 2 — Create a Named Session (Priority: P2)

A developer wants to start a fresh Claude session for a specific project. They choose "new session" from the picker, provide a mandatory name, and `cs` creates a dedicated tmux session (in the cs socket directory), starts Claude inside it, and attaches the terminal.

**Why this priority**: Named sessions allow parallel workstreams (e.g., one session per repo or task) without confusion.

**Independent Test**: Create two sessions with distinct names, detach from each, run `cs` and verify both are listed with their names and working directories.

**Acceptance Scenarios**:

1. **Given** the user selects "new session", **When** they provide a session name, **Then** the session is created with that exact name and Claude launches inside it.
2. **Given** the user selects "new session" but provides no name, **Then** the tool does not proceed — it prompts again until a name is supplied.
3. **Given** a session with the requested name already exists, **When** the user tries to create it, **Then** the tool informs them and switches to the attach flow for that session instead.

---

### User Story 3 — Delete a Stale Session (Priority: P3)

A developer notices a dead or irrelevant Claude session in the list. From the picker, they can mark it for deletion without dropping into tmux commands.

**Why this priority**: Session hygiene matters over time; a growing list of dead sessions degrades the picker UX.

**Independent Test**: Create and kill a session externally, verify it still appears in the list, then use the delete action and confirm it is gone.

**Acceptance Scenarios**:

1. **Given** a session is listed in the picker, **When** the user triggers the delete action (e.g., a keybinding), **Then** the session is removed after confirmation.
2. **Given** the user cancels deletion, **Then** the session remains intact.

---

### Edge Cases

- What happens when `cs setup` has not been run and a dependency is missing?
- If the user is already inside any tmux session, `cs` exits with a message to detach first — it never nests sessions.
- What happens if the working directory stored for a session no longer exists?
- A session whose Claude process has exited is shown in the picker with a `[dead]` indicator; the user can attach to inspect, restart Claude manually, or delete it — the tool never auto-removes it.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `cs` binary MUST be invokable from any macOS shell without arguments as the primary usage pattern.
- **FR-002**: The tool MUST manage all its tmux sessions through a dedicated socket directory (e.g., `~/.cs/sockets/`) so they are fully isolated from the user's personal tmux sessions.
- **FR-003**: When existing sessions are present, the tool MUST display a fuzzy-searchable interactive picker with `[ + new session ]` as the first pre-selected entry, followed by all cs-managed sessions; no manually created tmux sessions outside the cs socket are shown.
- **FR-004**: Each session entry in the picker MUST show at minimum the session name and the working directory it was created from.
- **FR-005**: The user MUST be able to attach to an existing session by selecting it in the picker.
- **FR-006**: The user MUST be able to create a new session from the picker by providing a mandatory name; the tool MUST NOT proceed without one.
- **FR-007**: When creating a new session, the tool MUST start the `claude` process inside the new tmux session and attach the user to it automatically.
- **FR-008**: The tool MUST NOT require the user to know any tmux commands to perform attach, create, or delete operations.
- **FR-009**: A `cs setup` subcommand MUST check for all required dependencies (`tmux`, `claude`, `fzf`) and offer to install missing ones before `cs` is used.
- **FR-010**: If the user is already inside any tmux session when they run `cs`, the tool MUST refuse to proceed and display a clear message instructing them to detach first; it MUST NOT nest sessions.
- **FR-011**: The user MUST be able to delete a cs-managed session from within the picker interface without dropping to the shell.
- **FR-012**: If a session name already exists when the user attempts to create one, the tool MUST redirect to the attach flow for that session rather than failing silently or creating a duplicate.
- **FR-013**: The picker MUST visually distinguish sessions whose Claude process has exited (e.g., a `[dead]` tag); such sessions MUST remain listed and selectable — the tool MUST NOT auto-delete them.

### Key Entities

- **Session**: A tmux session created and managed by `cs`, running within the cs socket directory, identified by a user-supplied name and the working directory it was created from.
- **Socket Directory**: The dedicated tmux socket path (e.g., `~/.cs/sockets/`) that scopes all cs sessions away from the user's personal tmux environment.
- **Picker**: The interactive fuzzy-search interface through which the user selects, creates, or deletes sessions.
- **Claude Process**: The `claude` CLI process running inside a session's tmux window.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user unfamiliar with tmux can launch, resume, and switch between Claude sessions in under 30 seconds per action.
- **SC-002**: All cs-managed sessions are discoverable in the picker — zero sessions are silently lost or hidden.
- **SC-003**: 100% of common operations (create, attach, delete) complete without the user typing a single tmux command.
- **SC-004**: The picker renders and becomes interactive in under 1 second on a machine with up to 50 sessions.
- **SC-005**: `cs setup` completes dependency verification in under 5 seconds and surfaces actionable install instructions for any missing dependency.
- **SC-006**: cs sessions are invisible in the user's personal tmux environment (`tmux ls` outside the cs socket shows zero cs sessions).

## Assumptions

- The target platform is macOS with a standard shell (zsh or bash); Linux compatibility is a nice-to-have but not required for v1.
- `fzf`, `tmux`, and the `claude` CLI are or can be installed via Homebrew; `cs setup` will offer Homebrew-based installation.
- Multiple simultaneous Claude sessions are a real workflow need (e.g., one per repo or per task).
- The binary will be distributed as a single self-contained script or compiled binary installable via Homebrew or direct download.
- Terminal multiplexer knowledge should be entirely abstracted away — the tool is the interface.
- Users are expected to run `cs setup` once before first use; the main `cs` command does not re-run full dependency checks on every invocation.
