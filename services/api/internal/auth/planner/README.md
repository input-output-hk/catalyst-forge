Planner configuration

This package constructs the application-specific RBAC scope planner.
It defines:

- Rules per resource type (scope chains and optional extractor overrides)
- Optional path-based overrides layered on top of the base rule planner

Bootstrap should call `planner.Build()` and pass the result into `rbac.Config.Scopes`.

