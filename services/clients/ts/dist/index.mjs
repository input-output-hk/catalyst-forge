var __defProp = Object.defineProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};

// src/api/client.ts
import createClient from "openapi-fetch";

// src/api/autoAuth.ts
var InMemoryTokenStore = class {
  constructor() {
    this.token = null;
  }
  get() {
    return this.token;
  }
  set(token) {
    this.token = token;
  }
};
function readCookie(name) {
  if (typeof document === "undefined") return null;
  const cookies = document.cookie ? document.cookie.split("; ") : [];
  for (const c of cookies) {
    const idx = c.indexOf("=");
    const key = decodeURIComponent(idx > -1 ? c.substring(0, idx) : c);
    if (key === name) {
      const val = idx > -1 ? c.substring(idx + 1) : "";
      return decodeURIComponent(val);
    }
  }
  return null;
}
function createAutoAuthFetch(baseFetch, options) {
  const refreshPath = options.refreshPath ?? "/api/v1/auth/refresh";
  const csrfCookieName = options.csrfCookieName ?? "__Host-csrf_token";
  const csrfHeaderName = options.csrfHeaderName ?? "X-CSRF-Token";
  const credentials = options.credentials ?? "include";
  const refreshPolicy = options.refreshPolicy ?? "always";
  let refreshPromise = null;
  async function doRefresh() {
    if (refreshPromise) return refreshPromise;
    refreshPromise = (async () => {
      try {
        const csrf = readCookie(csrfCookieName);
        const headers = new Headers({ "content-type": "application/json" });
        if (csrf) headers.set(csrfHeaderName, csrf);
        const resp = await baseFetch(new URL(refreshPath, options.baseUrl).toString(), {
          method: "POST",
          headers,
          credentials,
          body: "{}"
        });
        if (!resp.ok) {
          options.tokenStore?.set(null);
          throw new Error(`refresh failed: ${resp.status}`);
        }
        try {
          const json = await resp.clone().json().catch(() => null);
          if (json && typeof json.access_token === "string") {
            options.tokenStore?.set(json.access_token);
          }
        } catch {
        }
      } finally {
        refreshPromise = null;
      }
    })();
    return refreshPromise;
  }
  return async (input, init) => {
    const originalUrl = typeof input === "string" || input instanceof URL ? input.toString() : input.url;
    const isRefreshCall = originalUrl.endsWith(refreshPath);
    const headers = new Headers(init?.headers || (typeof input !== "string" && !(input instanceof URL) ? input.headers : void 0));
    const token = options.tokenStore?.get() ?? null;
    const hadAuthHeader = Boolean(token);
    if (token) headers.set("Authorization", `Bearer ${token}`);
    const method = (init?.method || (typeof input !== "string" && !(input instanceof URL) ? input.method : "GET")).toUpperCase();
    if (method !== "GET" && method !== "HEAD") {
      const csrf = readCookie(csrfCookieName);
      if (csrf && !headers.has(csrfHeaderName)) headers.set(csrfHeaderName, csrf);
    }
    const attempt = async () => {
      return baseFetch(input, { ...init, headers, credentials });
    };
    let resp = await attempt();
    if (resp.status === 401 && !isRefreshCall) {
      const shouldRefresh = refreshPolicy === "always" ? true : refreshPolicy === "whenHeaderPresent" ? hadAuthHeader : false;
      if (shouldRefresh) {
        try {
          await doRefresh();
        } catch {
          return resp;
        }
        const method2 = (init?.method || (typeof input !== "string" && !(input instanceof URL) ? input.method : "GET")).toUpperCase();
        const body = init?.body;
        const replaySafe = !body || typeof body === "string" || method2 === "GET" || method2 === "HEAD";
        if (!replaySafe) return resp;
        const newToken = options.tokenStore?.get() ?? null;
        if (newToken) headers.set("Authorization", `Bearer ${newToken}`);
        resp = await attempt();
      }
    }
    return resp;
  };
}

// src/api/client.ts
var BearerTokenProvider = class _BearerTokenProvider {
  constructor(token) {
    this.token = token;
  }
  get middleware() {
    const token = this.token;
    return {
      async onRequest({ request }) {
        request.headers.set("Authorization", `Bearer ${token}`);
        return request;
      }
    };
  }
  /**
   * Create from environment variable
   */
  static fromEnv(envVar = "Forge_API_TOKEN") {
    const token = process.env[envVar] || process.env.Forge_TOKEN;
    if (!token) {
      throw new Error(`Environment variable ${envVar} or Forge_TOKEN not set`);
    }
    return new _BearerTokenProvider(token);
  }
};
var ApiKeyProvider = class _ApiKeyProvider {
  constructor(apiKey, headerName = "X-API-Key") {
    this.apiKey = apiKey;
    this.headerName = headerName;
  }
  get middleware() {
    const headerName = this.headerName;
    const apiKey = this.apiKey;
    return {
      async onRequest({ request }) {
        request.headers.set(headerName, apiKey);
        return request;
      }
    };
  }
  /**
   * Create from environment variable
   */
  static fromEnv(envVar = "Forge_API_KEY", headerName = "X-API-Key") {
    const apiKey = process.env[envVar];
    if (!apiKey) {
      throw new Error(`Environment variable ${envVar} not set`);
    }
    return new _ApiKeyProvider(apiKey, headerName);
  }
};
var BasicAuthProvider = class {
  constructor(username, password) {
    this.credentials = btoa(`${username}:${password}`);
  }
  get middleware() {
    const credentials = this.credentials;
    return {
      async onRequest({ request }) {
        request.headers.set("Authorization", `Basic ${credentials}`);
        return request;
      }
    };
  }
};
var errorHandlingMiddleware = {
  async onResponse({ response }) {
    if (!response.ok) {
      const contentType = response.headers.get("content-type");
      let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
      if (contentType?.includes("application/json")) {
        try {
          const errorBody = await response.clone().json();
          if (errorBody.error || errorBody.message) {
            errorMessage = errorBody.error || errorBody.message;
          }
        } catch {
        }
      }
      response.errorMessage = errorMessage;
    }
    return response;
  }
};
var loggingMiddleware = {
  async onRequest({ request }) {
    console.log(`[Forge API] ${request.method} ${request.url}`);
    return request;
  },
  async onResponse({ response, request }) {
    console.log(`[Forge API] ${request.method} ${request.url} -> ${response.status}`);
    return response;
  }
};
var ForgeClient = class _ForgeClient {
  constructor(options) {
    const middleware = options.middleware || [];
    middleware.push(errorHandlingMiddleware);
    const enableAutoAuth = options.autoAuth ?? typeof window !== "undefined";
    const tokenStore = options.tokenStore;
    this.tokenStore = enableAutoAuth ? tokenStore : void 0;
    const wrappedFetch = enableAutoAuth ? createAutoAuthFetch(options.fetch ?? fetch, {
      baseUrl: options.baseUrl,
      tokenStore,
      credentials: "include",
      // Refresh on any 401 (except the refresh call itself). This supports cookie-based auth
      // where no Authorization header is present but a refresh should still occur on expiry.
      refreshPolicy: "always"
    }) : options.fetch ?? fetch;
    this.client = createClient({
      baseUrl: options.baseUrl,
      headers: options.headers,
      fetch: wrappedFetch,
      // @ts-ignore - openapi-fetch middleware types are a bit different
      middleware
    });
    this.httpFetch = wrappedFetch;
    this.baseUrl = options.baseUrl;
  }
  /**
   * Get the underlying openapi-fetch client for direct use
   */
  get raw() {
    return this.client;
  }
  /**
   * Get the fetch function used by this client (includes AutoAuth if enabled)
   */
  getFetch() {
    return this.httpFetch;
  }
  /**
   * Get the base URL configured for this client
   */
  getBaseUrl() {
    return this.baseUrl;
  }
  /**
   * Create client with bearer token authentication
   */
  static withBearerToken(baseUrl, token, options) {
    const authProvider = new BearerTokenProvider(token);
    return new _ForgeClient({
      baseUrl,
      ...options,
      middleware: [authProvider.middleware, ...options?.middleware || []]
    });
  }
  /**
   * Create client with API key authentication
   */
  static withApiKey(baseUrl, apiKey, headerName = "X-API-Key", options) {
    const authProvider = new ApiKeyProvider(apiKey, headerName);
    return new _ForgeClient({
      baseUrl,
      ...options,
      middleware: [authProvider.middleware, ...options?.middleware || []]
    });
  }
  /**
   * Create client with basic authentication
   */
  static withBasicAuth(baseUrl, username, password, options) {
    const authProvider = new BasicAuthProvider(username, password);
    return new _ForgeClient({
      baseUrl,
      ...options,
      middleware: [authProvider.middleware, ...options?.middleware || []]
    });
  }
  /**
   * Create client from environment variables
   */
  static fromEnv(options) {
    const baseUrl = process.env.Forge_API_URL || "https://api.Forge.example.com";
    if (process.env.Forge_API_TOKEN || process.env.Forge_TOKEN) {
      const authProvider = BearerTokenProvider.fromEnv();
      return new _ForgeClient({
        baseUrl,
        ...options,
        middleware: [authProvider.middleware, ...options?.middleware || []]
      });
    }
    if (process.env.Forge_API_KEY) {
      const authProvider = ApiKeyProvider.fromEnv();
      return new _ForgeClient({
        baseUrl,
        ...options,
        middleware: [authProvider.middleware, ...options?.middleware || []]
      });
    }
    return new _ForgeClient({
      baseUrl,
      ...options
    });
  }
  /**
   * Perform health check
   */
  async healthCheck() {
    const response = await this.client.GET("/healthz");
    if (!response.response.ok) {
      throw new Error(`Health check failed: ${response.response.status}`);
    }
    return response.data;
  }
  /**
   * Helper to check if an error response is a specific status code
   */
  static isStatus(error, status) {
    if (error && typeof error === "object" && "response" in error) {
      const response = error.response;
      return response?.status === status;
    }
    return false;
  }
  static isUnauthorized(error) {
    return this.isStatus(error, 401);
  }
  static isForbidden(error) {
    return this.isStatus(error, 403);
  }
  static isNotFound(error) {
    return this.isStatus(error, 404);
  }
  static isConflict(error) {
    return this.isStatus(error, 409);
  }
  static isServerError(error) {
    if (error && typeof error === "object" && "response" in error) {
      const response = error.response;
      return response?.status >= 500 && response?.status < 600;
    }
    return false;
  }
  /**
   * Set the current access token (used by AutoAuth wrapper)
   */
  setAccessToken(token) {
    this.tokenStore?.set(token);
  }
  /**
   * Get the current access token
   */
  getAccessToken() {
    return this.tokenStore?.get() ?? null;
  }
};

// src/api/webauthn.ts
var webauthn_exports = {};
__export(webauthn_exports, {
  login: () => login,
  registerCredential: () => registerCredential,
  stepUp: () => stepUp
});
function b64urlToBuf(value) {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const pad = base64.length % 4 ? 4 - base64.length % 4 : 0;
  const b64 = base64 + "=".repeat(pad);
  const str = typeof atob === "function" ? atob(b64) : Buffer.from(b64, "base64").toString("binary");
  const bytes = new Uint8Array(str.length);
  for (let i = 0; i < str.length; i++) bytes[i] = str.charCodeAt(i);
  return bytes.buffer;
}
function bufToB64url(buf) {
  const bytes = new Uint8Array(buf);
  let str = "";
  for (let i = 0; i < bytes.length; i++) str += String.fromCharCode(bytes[i]);
  const b64 = typeof btoa === "function" ? btoa(str) : Buffer.from(str, "binary").toString("base64");
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}
function decodeRequestOptions(options) {
  const src = options?.publicKey ?? options;
  const out = { ...src };
  if (Array.isArray(out.allowCredentials)) {
    out.allowCredentials = out.allowCredentials.map((c) => ({
      ...c,
      id: typeof c.id === "string" ? b64urlToBuf(c.id) : c.id
    }));
  }
  if (typeof out.challenge === "string") out.challenge = b64urlToBuf(out.challenge);
  return out;
}
function decodeCreationOptions(options) {
  const src = options?.publicKey ?? options;
  const out = { ...src };
  if (typeof out.challenge === "string") out.challenge = b64urlToBuf(out.challenge);
  if (out.user && typeof out.user.id === "string") out.user = { ...out.user, id: b64urlToBuf(out.user.id) };
  if (Array.isArray(out.excludeCredentials)) {
    out.excludeCredentials = out.excludeCredentials.map((c) => ({
      ...c,
      id: typeof c.id === "string" ? b64urlToBuf(c.id) : c.id
    }));
  }
  return out;
}
function encodeAssertion(cred) {
  const assertion = cred;
  return {
    id: cred.id,
    type: cred.type,
    rawId: bufToB64url(cred.rawId),
    response: {
      clientDataJSON: bufToB64url(assertion.response.clientDataJSON),
      authenticatorData: bufToB64url(assertion.response.authenticatorData),
      signature: bufToB64url(assertion.response.signature),
      userHandle: assertion.response.userHandle ? bufToB64url(assertion.response.userHandle) : void 0
    }
  };
}
function encodeAttestation(cred) {
  const attestation = cred;
  return {
    id: cred.id,
    type: cred.type,
    rawId: bufToB64url(cred.rawId),
    response: {
      clientDataJSON: bufToB64url(attestation.response.clientDataJSON),
      attestationObject: bufToB64url(attestation.response.attestationObject),
      transports: typeof attestation.response.getTransports === "function" ? attestation.response.getTransports() : void 0
    }
  };
}
async function login(client) {
  if (typeof navigator === "undefined" || !navigator.credentials) throw new Error("WebAuthn not available");
  const raw = client.raw;
  const begin = await raw.POST("/api/v1/auth/login/begin");
  if (!begin.response.ok) throw new Error("login begin failed");
  const { publicKey, session_key } = begin.data || {};
  const cred = await navigator.credentials.get({ publicKey: decodeRequestOptions(publicKey) });
  const complete = await raw.POST("/api/v1/auth/login/complete", {
    body: { session_key, credential: encodeAssertion(cred) }
  });
  if (!complete.response.ok) throw new Error("login complete failed");
}
async function registerCredential(client, deviceName) {
  if (typeof navigator === "undefined" || !navigator.credentials) throw new Error("WebAuthn not available");
  const raw = client.raw;
  const begin = await raw.POST("/api/v1/auth/credentials/add/begin", { body: { device_name: deviceName } });
  if (!begin.response.ok) throw new Error("register begin failed");
  const { publicKey, session_key } = begin.data || {};
  const cred = await navigator.credentials.create({ publicKey: decodeCreationOptions(publicKey) });
  const complete = await raw.POST("/api/v1/auth/credentials/add/complete", {
    body: { session_key, credential: encodeAttestation(cred) }
  });
  if (!complete.response.ok) throw new Error("register complete failed");
}
async function stepUp(client) {
  if (typeof navigator === "undefined" || !navigator.credentials) throw new Error("WebAuthn not available");
  const raw = client.raw;
  const begin = await raw.POST("/api/v1/auth/step-up/begin");
  if (!begin.response.ok) throw new Error("step-up begin failed");
  const { publicKey, session_key } = begin.data || {};
  const cred = await navigator.credentials.get({ publicKey: decodeRequestOptions(publicKey) });
  const complete = await raw.POST("/api/v1/auth/step-up/complete", {
    body: { session_key, credential: encodeAssertion(cred) }
  });
  if (!complete.response.ok) throw new Error("step-up complete failed");
}
export {
  ApiKeyProvider,
  BasicAuthProvider,
  BearerTokenProvider,
  ForgeClient,
  InMemoryTokenStore,
  createAutoAuthFetch,
  errorHandlingMiddleware,
  loggingMiddleware,
  webauthn_exports as webauthn
};
