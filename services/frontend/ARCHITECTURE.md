# Catalyst Forge – Architecture

## Assumptions
- MVP is design/UX-first with fully mocked data and flows.
- No backend; state is in-memory and resets on refresh.
- Flows are single-path with clear success; errors are simulated only.

## Open Questions
- Exact copy for confirmations and empty/error states?
- Required table columns and tags for filters?
- Any compliance notes for audit log retention formatting?

## Sitemap
- / (Dashboard)
- /services
- /environments
- /jobs
- /secrets
- /audit
- /settings
- /auth-demo

## Component Map
- Global shell: AppLayout (Sidebar + TopBar + FeatureFlagsPanel)
- CommandPalette (⌘K) using shadcn Command
- Feature flags: darkMode, compact, streamLogs
- Pages use AppStore for data and actions
- LogViewer for streaming logs

## Data Layer
- src/mocks/fixtures.ts – seed data
- src/mocks/latency.ts – 200–800ms latency
- src/store/app-store.tsx – React context store with actions

## Theming
- src/styles/tokens.css defines HSL tokens for colors, gradients, shadows, radii, fonts
- Tailwind uses Inter (UI) and JetBrains Mono (logs)

## Keyboard/Accessibility
- ⌘K opens Command Palette; Esc closes
- Visible focus rings via shadcn defaults

