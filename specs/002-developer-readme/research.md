# Research: Developer Onboarding & README

## Logo Format Decision

**Decision:** ASCII art logo embedded directly in README, plus an SVG badge/wordmark as a secondary asset.

**Rationale:** ASCII art renders identically in all GitHub themes (light/dark) with zero image hosting concerns, loads instantly, and fits the "well-worn shell alias" aesthetic of the constitution. An SVG wordmark can be added to `assets/logo.svg` for use in future blog posts or social sharing, but is optional. PNG is rejected because it requires binary asset management, has no dark-mode variant story, and doesn't match the CLI tool aesthetic.

**Alternatives considered:**
- PNG logo: Rejected — binary in repo, no automatic dark mode support, feels heavyweight for a CLI tool
- GitHub-hosted SVG: Acceptable but more complex; deferred to future iteration
- Shields.io badges only: Insufficient for visual identity; good complement but not a replacement for a logo

---

## README Structure Decision

**Decision:** Follow the "README for CLI tools" pattern used by projects like `gh`, `bat`, `ripgrep`, `fzf`:

1. ASCII logo / name header
2. One-line tagline
3. 2-3 sentence value proposition (the "why")
4. Demo (could be a terminal GIF or a static code block showing a session list)
5. Quick-start (install + first run, ≤5 steps)
6. Concept / how it works (the mental model section)
7. Usage reference (command table)
8. Platform & prerequisites
9. Contributing (one paragraph)

**Rationale:** This structure front-loads the value proposition and lowers cognitive load for scanners. Successful CLI tools (fzf, bat, gh) use this exact ordering — the "aha moment" (demo or clear tagline) comes before installation instructions.

**Alternatives considered:**
- Full-page prose introduction: Rejected — developers scan, they don't read; the demo/tagline must be above the fold
- Single long usage guide: Rejected — conflates onboarding with reference; new users need a narrative, existing users need a reference

---

## Demo Asset Decision

**Decision:** Use a static `asciinema` terminal recording embedded as a link, OR a static terminal screenshot rendered as a code block inside the README. The code block approach is used for v1 (no external service dependency). A real recording can replace it later.

**Rationale:** A static code block showing a fake-but-realistic `cs` session list is sufficient to communicate the UX and requires no external tooling. Projects like `fzf` ship their README with just a GIF; `cs` can start with a code block and upgrade to a GIF/asciinema link in a future iteration.

**Alternatives considered:**
- Asciinema embed: Great UX but requires external service dependency; deferred
- Animated GIF: Ideal but requires screen recording tooling; deferred
- Static screenshot: Acceptable but loses the terminal aesthetic; rejected in favor of a styled code block

---

## Session Naming / Mental Model Framing

**Decision:** Frame the core concept as "you name the context, Claude doesn't." Use concrete, real-world session name examples that demonstrate focus:

- `auth-redesign` — working on a specific feature branch
- `write-release-notes` — a writing task distinct from coding
- `debug-api-timeout` — an investigation thread
- `explore-new-framework` — a learning spike

**Rationale:** The spec requires communicating that session themes are human-chosen. The examples above are intentionally diverse (feature work, writing, debugging, learning) to show that sessions map to *cognitive contexts*, not just code tasks.

---

## Tagline Decision

**Decision:** `cs — your sessions, your focus.`

Alternatives evaluated:
- "Multiple Claude sessions, zero tmux knowledge" — too functional, doesn't convey the ownership concept
- "Scale your Claude workflow" — vague
- "One command. Many contexts." — closer but doesn't mention ownership
- "Your sessions, your focus." — conveys the human-ownership and focus concept concisely

---

## Badges Decision

**Decision:** Include three badges: Go version, build status (placeholder for CI), and license. Keep badge count low; badge inflation is a common anti-pattern that adds noise.

---

## Contributing Section Decision

**Decision:** A single paragraph with a link to opening issues/PRs on GitHub. No formal CONTRIBUTING.md in this feature (out of scope per spec). The README contributing section sets expectations without creating documents that won't be maintained.
