# Feature Specification: Session Model & Effort Selection

**Feature Branch**: `003-session-model-effort`  
**Created**: 2026-04-21  
**Status**: Draft  
**Input**: User description: "Let us enhance our new session creator to add two extra steps - one for the model name and the other for the effort level. I often have sizing in mind when opening a session and that judgement is what we want to bring to center when selecting a session. Additionally, when displaying session names (for new sessions) let us also display the model and effort we started with and fetching env vars if needed to show the current state."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Select Model and Effort When Creating a Session (Priority: P1)

When a developer starts a new `cs` session, after providing the session name they are presented with two additional prompts: one to choose the Claude model and one to choose the effort level. The model selection defaults to the current `ANTHROPIC_MODEL` environment variable (or the system default if unset). The effort selection defaults to the current `CLAUDE_CODE_EFFORT_LEVEL` environment variable (or `medium` if unset). Both choices are made via the same fuzzy-searchable picker interaction already used elsewhere in `cs`.

**Why this priority**: Choosing the right model and effort is a sizing judgment the developer makes at the start of every session — putting it front and center during session creation surfaces that decision at exactly the right moment rather than requiring a separate configuration step afterward.

**Independent Test**: Create a new session, select `opus` as the model and `high` as effort, then verify that Claude launches with `ANTHROPIC_MODEL=opus` and `CLAUDE_CODE_EFFORT_LEVEL=high` set in the tmux session environment.

**Acceptance Scenarios**:

1. **Given** a developer is creating a new session and has provided a name, **When** the model prompt appears, **Then** a selectable list of available Claude models is shown and the current `ANTHROPIC_MODEL` value (or system default) is pre-selected.
2. **Given** the model prompt is shown, **When** the developer selects a model, **Then** the effort prompt is shown next with a list of effort levels and the current `CLAUDE_CODE_EFFORT_LEVEL` value (or `medium`) pre-selected.
3. **Given** the developer selects an effort level, **When** both choices are confirmed, **Then** Claude launches inside the new tmux session with the chosen model and effort level active for that session.
4. **Given** the developer presses Enter without changing either prompt's pre-selected value, **Then** the defaults are used and Claude launches with whatever the environment currently specifies — no extra configuration required.

---

### User Story 2 — Session List Shows Model and Effort (Priority: P1)

When a developer opens `cs` and existing sessions are shown in the picker, each session entry displays the model and effort level that were active when the session was created. If that information is not recorded for an older session, a clear fallback indicator is shown (e.g., `model: unknown`).

**Why this priority**: Seeing model and effort in the session list is the primary way the developer can make an informed choice about which session to resume — it brings the sizing context to the surface without requiring them to remember or check notes.

**Independent Test**: Create two sessions with different model/effort combinations, run `cs`, and verify the picker shows each session's model and effort alongside its name and directory.

**Acceptance Scenarios**:

1. **Given** one or more sessions exist, **When** the picker renders, **Then** each session entry includes both the model name and effort level that were set at creation time.
2. **Given** a session was created before this feature was added (no model/effort recorded), **When** the picker renders, **Then** the session entry shows a clear fallback such as `model: unknown, effort: unknown` rather than blank fields or an error.
3. **Given** the picker is displaying session metadata, **When** the developer scans the list, **Then** model and effort are legible alongside session name and working directory without requiring horizontal scrolling or truncation of important fields.

---

### User Story 3 — Current Environment State Reflected in Defaults (Priority: P2)

When a developer has exported `ANTHROPIC_MODEL` or `CLAUDE_CODE_EFFORT_LEVEL` before running `cs`, those values are automatically reflected as the pre-selected defaults in the model and effort prompts during new session creation. The developer does not need to re-enter values they have already set in their shell.

**Why this priority**: Developers who use shell aliases (e.g., `claude-opus`) already have environment variables set — respecting those values reduces redundant input and makes the tool feel integrated with their existing workflow.

**Independent Test**: Export `ANTHROPIC_MODEL=haiku` and `CLAUDE_CODE_EFFORT_LEVEL=low` in the shell, then run `cs` to create a new session. Verify that `haiku` and `low` are pre-selected in their respective prompts.

**Acceptance Scenarios**:

1. **Given** `ANTHROPIC_MODEL` is set in the current shell, **When** the model prompt appears, **Then** that model is pre-selected in the picker.
2. **Given** `CLAUDE_CODE_EFFORT_LEVEL` is set in the current shell, **When** the effort prompt appears, **Then** that effort level is pre-selected in the picker.
3. **Given** neither variable is set, **When** the prompts appear, **Then** sensible defaults are pre-selected (`sonnet` for model, `medium` for effort, or the system-wide default if configured).

---

### Edge Cases

- What happens when the developer creates a session using a shell alias (e.g., `claude-opus`) that sets env vars inline — will those env vars be visible to `cs` when it reads defaults? (Yes — inline env vars are in scope for the `cs` process when launched that way.)
- What if the listed model names change as Claude releases new models? The list of available models should be easy to update without modifying session logic.
- If a session's stored model or effort is no longer a valid option (e.g., a retired model), the picker still shows it as metadata rather than erroring.
- What if the developer wants to change the model or effort for an existing session after the fact? This is out of scope for this feature; it only covers creation-time selection and display.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: During new session creation, after the session name is confirmed, the tool MUST present a model selection prompt listing all supported Claude model options.
- **FR-002**: The model selection prompt MUST pre-select the value of `ANTHROPIC_MODEL` from the session creator's environment; if unset, a designated default model (e.g., `sonnet`) MUST be pre-selected.
- **FR-003**: After the model is selected, the tool MUST present an effort level selection prompt listing all supported effort levels (`low`, `medium`, `high`, `xhigh`).
- **FR-004**: The effort level prompt MUST pre-select the value of `CLAUDE_CODE_EFFORT_LEVEL` from the session creator's environment; if unset, `medium` MUST be pre-selected.
- **FR-005**: The chosen model and effort level MUST be set as environment variables within the new tmux session so that `claude` launches with those values active.
- **FR-006**: The model and effort level chosen at session creation time MUST be persisted alongside the session's other metadata (name, working directory).
- **FR-007**: The session picker MUST display the stored model and effort level for each existing session entry alongside the session name and working directory.
- **FR-008**: For sessions that predate this feature and have no stored model/effort, the picker MUST display a clear fallback indicator (e.g., `model: unknown`) rather than an empty field or an error.
- **FR-009**: Both the model and effort prompts MUST support the same fuzzy-search interaction as the rest of the `cs` picker interface — keyboard navigation, search-to-filter, and Enter-to-confirm.
- **FR-010**: If the developer accepts the pre-selected default on both prompts without changing anything, the session MUST launch with those defaults; no additional input MUST be required.

### Key Entities

- **Session Metadata**: The persisted record for a cs-managed session, now extended to include `model` (the Claude model alias) and `effort` (the effort level string) alongside existing fields (name, working directory, creation time).
- **Model Option**: One of the supported Claude model aliases (e.g., `sonnet`, `opus`, `haiku`, `opus[1m]`). The list is maintained in a single configurable location.
- **Effort Level**: One of the defined effort tiers (`low`, `medium`, `high`, `xhigh`), representing the thinking depth allocated to Claude during the session.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can complete the model and effort selection steps within 5 seconds on top of the existing session name prompt — the total new-session creation flow takes under 15 seconds end to end.
- **SC-002**: 100% of newly created sessions have model and effort metadata visible in the picker immediately after creation, with no manual refresh or restart required.
- **SC-003**: Sessions created before this feature was introduced continue to appear in the picker without errors; their model and effort fields show a clear fallback indicator rather than causing crashes or blank entries.
- **SC-004**: Pre-selection of environment-variable defaults means a developer who has already set `ANTHROPIC_MODEL` and `CLAUDE_CODE_EFFORT_LEVEL` can create a session with zero extra keystrokes beyond what was required before this feature.
- **SC-005**: The session list remains legible with model and effort metadata added — no truncation of session name or working directory occurs on a standard 80-column terminal.

## Assumptions

- The existing `cs` session metadata storage mechanism (however it is implemented) can be extended to include two additional string fields (`model`, `effort`) without breaking existing sessions.
- The supported model list is small enough (under 10 entries) to be hardcoded or stored in a simple configuration file; dynamic fetching from an API is out of scope.
- The tmux session environment is the correct mechanism for passing model and effort to the `claude` process; environment variables set on the tmux session will be inherited by all processes started inside it.
- Users are on macOS with zsh or bash; environment variable reading and tmux `set-environment` commands are available.
- The effort level vocabulary (`low`, `medium`, `high`, `xhigh`) matches the values accepted by `CLAUDE_CODE_EFFORT_LEVEL` as documented. `max` is excluded from the selectable list as it is session-scoped by nature and does not persist.
- Changing model or effort for an existing session after creation is explicitly out of scope for this feature — it is a future enhancement.
