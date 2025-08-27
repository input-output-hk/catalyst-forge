## 0) Quick mental model (how these parts fit)

* **Kratos** = where users authenticate (your Google sign‑in, passwords, MFA, etc.). It issues *sessions*; check them via `/sessions/whoami`. ([Ory Corp][1])
* **Hydra** = your OAuth2/OIDC Authorization Server. It **doesn’t** do user auth; it calls your **login URL** and **consent URL** (your app) to decide who the user is and what scopes to grant. It can also call a **token hook** you host to add custom claims at token mint‑time. ([Ory Corp][2])
* **Oathkeeper** = your PEP in front of the API. It authenticates requests (via cookie→Kratos, JWT→Hydra JWKS, or introspection) and then authorizes (either directly with Keto or by calling a **remote PDP**). ([Ory Corp][3])
* **Keto** = Zanzibar‑style relationship checks (“can X do Y on Z?”). ([GitHub][4])

---

## 1) The “Auth Orchestrator” service (one service, 3 endpoints)

> Implement as a small stateless HTTP service (Go/Node), exposed *internally* except for the login/consent UI routes.

### Endpoints

1. **`GET /oauth2/login`**
   **Purpose:** Handle Hydra’s `login_challenge`.
   **Flow:**

   * Read `login_challenge`. Call Hydra Admin: `getOAuth2LoginRequest`. If `skip=true`, **accept** immediately. ([Ory Corp][2])
   * Else check for a Kratos session:

     * If browser: call Kratos `/sessions/whoami` using cookies. If not authenticated, redirect to Kratos login UI with `return_to=/oauth2/login?login_challenge=...`. ([Ory Corp][1])
     * If CLI: you’ll use Authorization Code + **PKCE** in the system browser; Hydra still lands here, and you still check Kratos session (created after Google sign‑in). ([Ory Corp][5], [IETF Datatracker][6])
   * Once you have a Kratos session, **accept login** with Hydra Admin `acceptOAuth2LoginRequest` and set `subject = kratos.identity.id` (or your preferred stable user id). Hydra redirects the user/CLI to your **consent** URL next. ([Ory Corp][2])

2. **`GET|POST /oauth2/consent`**
   **Purpose:** Handle Hydra’s `consent_challenge`.
   **Flow:**

   * Read `consent_challenge`. Call Hydra Admin: `getOAuth2ConsentRequest`. Decide whether to show a screen or auto‑grant (for first‑party clients you can **skip consent**). ([Ory Corp][2])
   * On accept, **grant scopes/audience** and provide a **session object** that will become token claims (parts into `id_token` and into access token “ext” claims by default). ([Ory Corp][2])
   * Typical mappings:

     * `id_token`: stable user attributes (email, name) from Kratos traits.
     * `access_token`: machine‑readable authorization context for APIs (e.g., `org_id`, `entitlements`, `tenant`, `roles`).
   * Redirect with `redirect_to` back to Hydra’s `/oauth2/auth` path which then issues `code` to the client. ([Ory Corp][2])

3. **`POST /hydra/token-hook`**
   **Purpose:** **claim enrichment** at mint‑time for **all grant types** (Auth Code, Refresh, Client Credentials, and **JWT‑Bearer**). Hydra POSTs here **before** issuing tokens; your JSON response can add/override `session.id_token` / `session.access_token` custom claims. You can also **deny** issuance (HTTP 403). ([Ory Corp][7])
   **Notes:**

   * For **JWT‑Bearer** (e.g., GitHub Actions OIDC), Hydra validates the assertion against a **trusted issuer** you pre‑registered; your hook receives the original assertion under `request.payload.assertion` so you can pull repo/branch/environment claims and add normalized fields to the access token. (Hydra has already checked `iss/sub/aud/exp` against the trust relationship.) ([Ory Corp][8])
   * By default, custom access‑token claims go under `ext`. You can configure flattening if you prefer top‑level. ([Ory Corp][7])

### Minimal Hydra configuration pointers

* Set Hydra **login/consent URLs** to your service:
  `ory patch oauth2-config --replace '/urls/login="https://auth.yourco/login"' --replace '/urls/consent="https://auth.yourco/consent"'`. ([Ory Corp][2])
* Register the **token hook**:
  `ory patch oauth2-config --add '/oauth2/token_hook/url="https://auth.yourco/hydra/token-hook"'`. ([Ory Corp][7])
* For GitHub Actions, register a **JWT‑Bearer trusted issuer** (`token.actions.githubusercontent.com`) with its public keys and scope limits via `trustOAuth2JwtGrantIssuer`. ([Ory Corp][8])

### Suggested claim model (consistent across human & machine)

* **Human (CLI)** tokens:
  `sub = user:<kratos-id>`; `ext: { email, orgs, roles, mfa_level }`
* **GitHub Actions** tokens (JWT‑Bearer):
  You **cannot** change `sub` in the hook, but you can normalize GH claims into your `ext` namespace, e.g.
  `ext: { gh_repository, gh_ref, gh_sha, gh_actor, gh_environment }` extracted from the GH assertion. (GH publishes the claim set and allows customizing `sub` templates.) ([GitHub Docs][9])

> Your APIs and PDP can then authorize on a single, predictable set of fields regardless of how the token was minted.

---

## 2) End‑to‑end: CLI flow (Auth Code + PKCE) with the middleware

1. CLI opens the system browser at `Hydra /oauth2/auth?client_id=cli&response_type=code&scope=openid%20offline&redirect_uri=http://127.0.0.1:NNNN/callback&code_challenge=...` (PKCE). ([IETF Datatracker][6], [Ory Corp][5])
2. Hydra checks if the user is logged in; if not, **redirects to** your `/oauth2/login?login_challenge=...`. Your app checks Kratos `/sessions/whoami`; if no session, it **redirects to Kratos login** (which in turn redirects to Google). ([Ory Corp][1])
3. After Google → Kratos, you have a Kratos session. Your app **accepts the Hydra login** (subject = kratos id). Hydra now **redirects to** your `/oauth2/consent?consent_challenge=...`. ([Ory Corp][2])
4. Your app **accepts consent**, adds any default claims (e.g., org/tenant), returns `redirect_to`. ([Ory Corp][2])
5. Hydra issues an **auth code** to the CLI’s loopback redirect. CLI exchanges code (with verifier). **Before** issuing tokens, Hydra calls your **`/hydra/token-hook`**, and you enrich claims (e.g., add RBAC, tenancy). Hydra returns tokens. ([Ory Corp][7])
6. CLI calls your API via Oathkeeper. Oathkeeper authenticates:

   * If you use **JWT access tokens**, configure the `jwt` authenticator to trust Hydra’s JWKS (fast path). ([Ory Corp][3])
   * If you use **opaque tokens**, use `oauth2_introspection` against Hydra (`/oauth2/introspect`). (Meant for internal usage; don’t expose publicly.) ([Ory Corp][10])

---

## 3) How the same service handles GitHub Actions (JWT‑Bearer)

1. Workflow calls Hydra `/oauth2/token` with `grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=<github-oidc-jwt>`; Hydra validates against your **trusted issuer** registration (signature, `iss/sub/aud/exp` + optional scope). ([Ory Corp][8])
2. **Hydra calls your `/hydra/token-hook`** with the parsed request. For JWT‑Bearer, the raw assertion is in `request.payload.assertion`. You parse claims like `repository`, `ref`, `actor`, `environment` and **copy them into `session.access_token`** (e.g., normalized `ext.*` fields). ([Ory Corp][7], [GitHub Docs][9])
3. You optionally **deny** issuance if the repo/org isn’t in an allowlist (respond 403). Otherwise return 200 with enriched claims. Hydra issues the token; Oathkeeper verifies it via `jwt` authenticator. ([Ory Corp][7])

---

## 6) Implementation checklist (pragmatic)

**Auth Orchestrator**

* **Hydra Admin access** (cluster‑internal only).
* **Routes**: `/oauth2/login`, `/oauth2/consent`, `/hydra/token-hook`.
* **Kratos**: call `/sessions/whoami` to identify the user after Google sign‑in. ([Ory Corp][1])
* **Claims library**: one place to map Kratos identity + GH OIDC → normalized `ext.*` fields.
* **Hardening**:

  * Protect token hook (Hydra→hook) with **API key** or **mTLS** as supported by Hydra token‑hook config. ([Ory Corp][7])
  * CSRF on HTML forms for login/consent UI (if you show UI).
  * Strict input validation of `*_challenge`.
* **Observability**: log and propagate `login_challenge` / `consent_challenge` as correlation IDs.

**Oathkeeper**

* **Authenticators**:

  * **`cookie_session`** → Kratos (for browser APIs). ([Ory Corp][3])
  * **`jwt`** → Hydra JWKS (for CLI & CI). ([Ory Corp][3])
  * (If you stick with opaque tokens, use `oauth2_introspection` → Hydra.) ([Ory Corp][10])
* **Authorizer**: `remote_json` → PDP. ([Ory Corp][11])

---

## 7) A few gotchas to avoid

* **Don’t expect Kratos to mint your OAuth tokens.** Kratos issues sessions; Hydra mints OAuth/OIDC tokens. Your app stitches them together via the login/consent flow. ([Ory Corp][2])
* **You can’t change `sub` in the token hook.** Map identities into your own `ext.*` (or flatten) and let PDP/Oathkeeper read those. ([Ory Corp][7])
* **Opaque vs JWT tokens.** JWTs + `jwt` authenticator are fastest (no network round‑trip per request). Introspection is fine for internal use only. ([Ory Corp][10])

---

[1]: https://www.ory.sh/docs/kratos/session-management/overview "Overview of sessions, Ory Session Cookies, and Ory Session Tokens | Ory"
[2]: https://www.ory.sh/docs/oauth2-oidc/custom-login-consent/flow "User login and consent flow | Ory"
[3]: https://www.ory.sh/docs/oathkeeper/pipeline/authn "Authenticators | Ory"
[4]: https://github.com/ory/keto?utm_source=chatgpt.com "Ory Keto"
[5]: https://www.ory.sh/docs/oauth2-oidc/authorization-code-flow?utm_source=chatgpt.com "OAuth2 authorization code flow"
[6]: https://datatracker.ietf.org/doc/html/rfc8252?utm_source=chatgpt.com "RFC 8252 - OAuth 2.0 for Native Apps"
[7]: https://www.ory.sh/docs/hydra/guides/claims-at-refresh "Customize claims with OAuth2 webhooks | Ory"
[8]: https://www.ory.sh/docs/hydra/guides/jwt "JSON Web Token (JWT) profile for OAuth2 | Ory"
[9]: https://docs.github.com/en/actions/concepts/security/openid-connect?utm_source=chatgpt.com "OpenID Connect"
[10]: https://www.ory.sh/docs/hydra/guides/oauth2-token-introspection?utm_source=chatgpt.com "OAuth 2.0 token introspection"
[11]: https://www.ory.sh/docs/oathkeeper/pipeline/authz "Authorizers | Ory"
[12]: https://github.com/ory/oathkeeper/issues/945?utm_source=chatgpt.com "Implementation of keto_engine_acp_ory for keto v0.6+ #945"
[13]: https://www.ory.sh/docs/oathkeeper/pipeline?utm_source=chatgpt.com "Access rule pipeline"
