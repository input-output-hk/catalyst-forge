# UX Guidelines

- Minimal, fast, keyboard-first.
- Large, confident headings; readable body text.
- Sticky table headers; quick filters on Audit.
- Obvious states for loading/empty/error; demo uses latency to showcase.
- Secrets: masked by default; explicit Reveal & Rotate flows, both confirm and log to Audit (mocked).
- Jobs: streaming logs with Pause/Download, monospaced font.
- Feature Flags panel toggles Dark mode, Compact density and Stream Logs.
- Command Palette (⌘K) for navigation.

## Empty/Error templates
- Empty: short explainer + primary action + link to docs (placeholder)
- Error: readable message, Retry, and Copy error details

## Demo Script (5 minutes)
1) Start on Dashboard; open Flags to toggle Dark mode.
2) ⌘K to navigate to Services, show sticky header.
3) Go to Jobs; pick running job; show logs streaming, pause, download.
4) Open Secrets; Reveal a secret; Rotate it and note Audit entry.
5) Open Audit Log; type filters.
6) Open Settings; add/remove a credential.
7) Auth Flows page; click through each step text.
