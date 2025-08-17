## Frontend modernization checklist (per-file)

Use this checklist to update any source file still using legacy patterns. Apply steps top-to-bottom for each file you touch.

### 1) Imports and client usage
- [ ] Replace ad-hoc API helpers with the generated client
  - Remove imports of `apiFetch`, `getApiBaseUrl`, and any direct `fetch` calls for spec-covered endpoints
  - Ensure the forge client import is used:
    ```ts
    import { forge } from "@/lib/client";
    ```
- [ ] Prefer typed OpenAPI calls
  - For reads:
    ```ts
    await forge.raw.GET("/api/v1/…", { params: { query: { … }, path: { … } } });
    ```
  - For writes:
    ```ts
    await forge.raw.POST("/api/v1/…", { body: { … } });
    ```

### 2) Types (no inline assertions)
- [ ] Import generated schema types:
  ```ts
  import type { paths, components } from "forge-client";
  ```
- [ ] Replace `as any` / `as unknown as` and hand-written shapes with generated types
  - Responses:
    ```ts
    type Res = paths["/api/v1/route"]["get"]["responses"][200]["content"]["application/json"];
    const data = res.data as Res;
    ```
  - Bodies:
    ```ts
    // Use components["schemas"]["…"] when request bodies are defined
    ```
- [ ] If types don’t match current payloads, fix the API Swagger docs, regenerate, then update code (don’t paper over with `any`).

### 3) Query and path parameters
- [ ] Do not build query strings manually. Use `params.query`/`params.path`:
  ```ts
  await forge.raw.GET("/api/v1/admin/audit", {
    params: { query: { actor_id, types, limit: 200 } },
  });
  ```

### 4) Auth, cookies, and headers
- [ ] Do not set `Authorization` manually; don’t persist tokens yourself
  - Rely on `forge` AutoAuth + cookies; no manual `refresh` preflight
- [ ] Do not hand-add CSRF headers; the client attaches them for mutating requests

### 5) Hard-coded URLs
- [ ] Remove `new URL("/api/v1/…", getApiBaseUrl())`, `fetch("/api/v1/…")`, and string-concatenated paths
- [ ] Use only the generated routes: `forge.raw.METHOD("/api/v1/…", …)`

### 6) React patterns
- [ ] Prefer feature API wrappers (e.g., `src/features/**/api/queries.ts`) over inline effects
- [ ] Use React Query for server data where appropriate

### 7) Error handling
- [ ] Check `res.response.ok`; on failure, read `res.response.text()` for details
- [ ] Surface errors via the unified toast system (not console-only)

### 8) If an endpoint is missing or params aren’t typed
- [ ] Update backend Swagger annotations on the handler (see `services/api/internal/api/handlers/**/swagger_docs.go`)
- [ ] Regenerate types and vendor the client (Earthly-only flow):
  ```bash
  # Rebuild Swagger
  cd services/api && earthly +swagger

  # Build TS client + types and export vendor artifact
  cd ../clients/ts && earthly +vendor

  # Vendor into frontend
  cd ../../frontend && earthly +vendor-client
  ```

### 9) Quick grep to find legacy patterns (run at `services/frontend`)
- [ ] `rg "apiFetch\(" src` — replace with `forge.raw.*`
- [ ] `rg "getApiBaseUrl\(" src` — remove; base URL is handled by client
- [ ] `rg "forgeFetch\(" src` — replace with `forge.raw.*`
- [ ] `rg "['\"]\/api\/v1\/(.+)['\"]" src` — replace hard-coded paths with client calls
- [ ] `rg "as any|as unknown as" src` — replace with generated types

### 10) Example: convert a GET with query and typed response
```ts
import { forge } from "@/lib/client";
import type { paths } from "forge-client";

type AuditRes = paths["/api/v1/admin/audit"]["get"]["responses"][200]["content"]["application/json"];

const res = await forge.raw.GET("/api/v1/admin/audit", {
  params: { query: { actor_id, types, limit: 200 } },
});
if (res.response.ok) {
  const data = res.data as AuditRes;
  // use data.events
}
```

---

What “done” looks like in a file
- No `apiFetch`, `getApiBaseUrl`, hard-coded `/api/v1/…` fetches, or `forgeFetch`
- All requests go through `forge.raw.*` with typed `params`/`body`
- No `any`/double-casts; responses and bodies use generated types
- No manual auth/CSRF handling
- Errors surfaced consistently


