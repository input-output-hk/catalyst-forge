At a high level we’ll use a standard **device‑code linking flow** (OAuth “device grant” style) so the CLI never has to do WebAuthn. The browser does the strong auth; the CLI just links to that session and gets a normal refresh/access token pair. This plays nicely with your existing “X‑CLI” header allowance and Authorization handling in the router today.&#x20;

---

# What we add (small + composable)

## New endpoints (all prefixed `/api/v1`)

**Device link (for login + step‑up via browser)**

* **POST `/api/v1/auth/device-link/begin`**
  Request (CLI): `{ "device_name": "Paul’s MBP • Forge CLI", "purpose": "login" }`
  Response:

  ```json
  {
    "device_code": "bGxY...x2",            // opaque, 10 min TTL
    "user_code": "J7FQ-K9",               // short, human-friendly
    "verification_uri": "https://app.example.com/cli/link",
    "verification_uri_complete": "https://app.example.com/cli/link?c=J7FQ-K9",
    "expires_in": 600,
    "interval": 5
  }
  ```

* **POST `/api/v1/auth/device-link/authorize`**
  Request (Browser, authenticated): `{ "device_code": "bGxY...x2" }`
  Behavior: requires a **fresh WebAuthn step‑up** (UV required). If step‑up not present, return a hint so the page can run your existing `/auth/step-up/begin|complete` ceremony, then retry authorize.

* **POST `/api/v1/auth/device-link/exchange`**
  Request (CLI): `{ "device_code": "bGxY...x2" }`
  Responses:

  * `authorization_pending` (keep polling every `interval` seconds)
  * `slow_down` (increase interval)
  * `expired_token` (start over)
  * **200 OK** with tokens:

    ```json
    {
      "access_token": "<jwt>",
      "expires_in": 1800,
      "refresh_token": "<opaque>",
      "refresh_expires_in": 2592000,
      "device_id": "uuid",
      "user": { "id": "uuid", "email": "you@dev.com", "roles": ["..."] }
    }
    ```

**Refresh (shared with browser; 1 small addition for CLI)**

* **POST `/api/v1/auth/refresh`**

  * Browser mode: refresh cookie + CSRF (unchanged).
  * **CLI mode:** send `X-CLI: 1` and either `Authorization: Refresh <opaque>` **or** JSON body `{ "refresh_token": "<opaque>" }`. CSRF not required in CLI mode.
    Response: `{ "access_token": "<jwt>" }` (+ rotated `refresh_token` if you choose to return it).

> Rationale: you already allow the `X-CLI` header in CORS; using it to signal “CLI path” lets the same endpoint serve both worlds without special handler logic.&#x20;

---

# Developer UX (what `forge` CLI does)

### `forge auth login`

1. CLI calls `/auth/device-link/begin` → prints `user_code` and opens `verification_uri_complete`.
2. User completes **WebAuthn in the browser** (your existing login/step‑up), then the browser page POSTs `/auth/device-link/authorize`.
3. CLI polls `/auth/device-link/exchange` until it gets tokens.
4. CLI stores the **refresh token** in the OS keychain and keeps the **access token** in memory.

### Every command thereafter

* Send `Authorization: Bearer <access_token>`.
* On `401` or near expiry, call `/auth/refresh` with `X-CLI: 1` and the stored refresh token.
* On **reuse detection** (stolen refresh), your server revokes the family; the CLI prints “session expired, please run `forge auth login`” and exits.

### `forge auth logout`

* CLI calls `/auth/logout` to revoke its current refresh token (you already have logout endpoints); then deletes its keychain entry.&#x20;

---

# Step‑up from the CLI (no new ceremony for users)

When a CLI call hits a **step‑up‑gated** endpoint (e.g., deploy, cert sign), your policy enforcer returns:

```json
HTTP 428
{
  "error": "step_up_required",
  "verification_uri": "https://app.example.com/cli/confirm",
  "hint": "Open the URL to confirm with your passkey."
}
```

CLI prints:
“🔒 This action requires confirmation. Opening browser… (or visit: …)”
User completes WebAuthn in the browser; your page calls existing `/auth/step-up/begin|complete`, then **(re)uses** `/auth/device-link/authorize` with the CLI’s **current** `device_code` (or the CLI can create a new one with `purpose: "step_up"`). The server writes a **step‑up grant bound to `device_id`** for 5 minutes. CLI retries the call; it now passes.

> No new cryptography or CLI‑specific auth—just reusing the **same step‑up primitive** and a device‑bound grant.

---

# Storage & model reuse

* **Device record** (reuse your “device management” concepts): `device_id`, `user_id`, `device_name`, `created_at`, `last_used_at`. Show these in your existing `/auth/devices` UI so users can revoke a CLI device like any other.&#x20;
* **Refresh token family** bound to `device_id`. Rotation + reuse‑detection unchanged.
* **Audit events**: `device_link_begin`, `device_link_authorized`, `device_link_exchange`, `refresh_reuse_detected`.

---

# Security notes (kept tight, minimal code)

* **Step‑up required** to authorize any device link. (Attackers with only email access can’t link a CLI.)
* **Short TTL** for `device_code` (10 min), polling interval 5s; rate‑limit per code and per IP.
* **Refresh token handling**:

  * Browser: cookie + CSRF (unchanged).
  * CLI: header/body (no CSRF) **only** on `/auth/refresh`; all other routes still use Bearer **access** tokens.
* **Scopes/claims**: same roles/permissions as the user; optionally add `"amr": ["webauthn","device_link"]` in the access JWT so you can trace how the session was established.
* **Revocation**: deleting a CLI device revokes its refresh family immediately; the CLI will be forced to re‑login.

---

# Minimal implementation checklist (fits your startup constraints)

1. **Add the three device‑link endpoints** and reuse your WebAuthn + step‑up services internally.
2. **Teach `/auth/refresh`** to accept `X-CLI: 1` + header/body `refresh_token` (browser path unchanged).&#x20;
3. **Add device table** if you don’t already have one (you do in the current codebase—can be adapted).&#x20;
4. **Wire policy**: no changes—CLI uses the same access JWT; step‑up works via the same enforcer.
5. **CLI commands**: `auth login`, `auth status`, `auth logout` (open browser with the complete verification URL; poll; store tokens in keychain).

---

# Optional polish (fast wins)

* Accept a `--device-name` flag; default to `<hostname> • Forge CLI`.
* If `open` isn’t available, just print the URL with the user code; don’t block.
* Friendly errors on `authorization_pending`, `expired_token`, and `reuse_detected`.

This gives developers a **two‑command** experience that feels native in the terminal, while you keep one unified auth model on the server—no fragile, handler‑level special cases. And it slots cleanly into your existing router + CORS/header posture (note your current `X‑CLI` header and Authorization handling).&#x20;
