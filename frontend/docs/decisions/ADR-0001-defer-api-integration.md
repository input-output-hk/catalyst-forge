# ADR-0001: Defer real API integration

Date: 2025-08-10

## Status

Accepted

## Context

We are building the frontend quickly with clear UX and mocked data. The backend API and auth flows are evolving in parallel.

## Decision

- Implement a mocked auth flow and use deterministic mock data services.
- Generate API types and wire a tiny client later (openapi-typescript + openapi-fetch), but not in this phase.
- Keep interfaces narrow and separate UI from data access to ease later integration.

## Consequences

- Faster UI progress and E2E coverage now.
- Minimal refactors later to switch mocked services to real client calls.
