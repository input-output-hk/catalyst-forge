export interface TokenStore {
  get(): string | null;
  set(token: string | null): void;
}

export class InMemoryTokenStore implements TokenStore {
  private token: string | null = null;
  get(): string | null {
    return this.token;
  }
  set(token: string | null): void {
    this.token = token;
  }
}

export interface AutoAuthOptions {
  baseUrl: string;
  refreshPath?: string; // default: /api/v1/auth/refresh
  csrfCookieName?: string; // default: __Host-csrf_token
  csrfHeaderName?: string; // default: X-CSRF-Token
  tokenStore?: TokenStore;
  credentials?: RequestCredentials; // default: include
  isBrowser?: boolean; // allow overriding for tests
  // Refresh behavior control:
  // - "always": existing behavior – refresh on any 401 (except refresh itself)
  // - "whenHeaderPresent": refresh only if an Authorization header was attached (bearer mode)
  // - "never": never auto-refresh
  refreshPolicy?: 'always' | 'whenHeaderPresent' | 'never';
}

function readCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const cookies = document.cookie ? document.cookie.split('; ') : [];
  for (const c of cookies) {
    const idx = c.indexOf('=');
    const key = decodeURIComponent(idx > -1 ? c.substring(0, idx) : c);
    if (key === name) {
      const val = idx > -1 ? c.substring(idx + 1) : '';
      return decodeURIComponent(val);
    }
  }
  return null;
}

export function createAutoAuthFetch(baseFetch: typeof fetch, options: AutoAuthOptions): typeof fetch {
  const refreshPath = options.refreshPath ?? '/api/v1/auth/refresh';
  const csrfCookieName = options.csrfCookieName ?? '__Host-csrf_token';
  const csrfHeaderName = options.csrfHeaderName ?? 'X-CSRF-Token';
  const credentials = options.credentials ?? 'include';
  const refreshPolicy = options.refreshPolicy ?? 'always';
  let refreshPromise: Promise<void> | null = null;

  async function doRefresh(): Promise<void> {
    if (refreshPromise) return refreshPromise;
    refreshPromise = (async () => {
      try {
        const csrf = readCookie(csrfCookieName);
        const headers = new Headers({ 'content-type': 'application/json' });
        if (csrf) headers.set(csrfHeaderName, csrf);
        const resp = await baseFetch(new URL(refreshPath, options.baseUrl).toString(), {
          method: 'POST',
          headers,
          credentials,
          body: '{}',
        });
        if (!resp.ok) {
          // Clear token on failed refresh (if store present)
          options.tokenStore?.set(null);
          throw new Error(`refresh failed: ${resp.status}`);
        }
        // Try to parse { access_token, expires_in }
        try {
          const json = await resp.clone().json().catch(() => null as any);
          if (json && typeof json.access_token === 'string') {
            options.tokenStore?.set(json.access_token);
          }
        } catch {
          // ignore JSON parse failures
        }
      } finally {
        // Important: unset the promise for next cycle without self-awaiting
        refreshPromise = null;
      }
    })();
    return refreshPromise;
  }

  return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const originalUrl = typeof input === 'string' || input instanceof URL ? input.toString() : input.url;
    const isRefreshCall = originalUrl.endsWith(refreshPath);

    const headers = new Headers(init?.headers || (typeof input !== 'string' && !(input instanceof URL) ? (input.headers as any) : undefined));
    const token = options.tokenStore?.get() ?? null;
    const hadAuthHeader = Boolean(token);
    if (token) headers.set('Authorization', `Bearer ${token}`);
    // CSRF for all mutating requests (server policy may require beyond refresh)
    const method = (init?.method || (typeof input !== 'string' && !(input instanceof URL) ? (input as Request).method : 'GET')).toUpperCase();
    if (method !== 'GET' && method !== 'HEAD') {
      const csrf = readCookie(csrfCookieName);
      if (csrf && !headers.has(csrfHeaderName)) headers.set(csrfHeaderName, csrf);
    }

    const attempt = async (): Promise<Response> => {
      return baseFetch(input, { ...init, headers, credentials });
    };

    let resp = await attempt();
    if (resp.status === 401 && !isRefreshCall) {
      const shouldRefresh = (
        refreshPolicy === 'always' ? true :
          refreshPolicy === 'whenHeaderPresent' ? hadAuthHeader :
            false
      );
      if (shouldRefresh) {
        try {
          await doRefresh();
        } catch {
          return resp; // propagate original 401 if refresh fails
        }
        // Retry only if body is safe to replay (no stream). Allow when method is GET/HEAD or body is string/undefined
        const method = (init?.method || (typeof input !== 'string' && !(input instanceof URL) ? (input as Request).method : 'GET')).toUpperCase();
        const body = init?.body;
        const replaySafe = !body || typeof body === 'string' || method === 'GET' || method === 'HEAD';
        if (!replaySafe) return resp;
        // Update Authorization header with possibly new token
        const newToken = options.tokenStore?.get() ?? null;
        if (newToken) headers.set('Authorization', `Bearer ${newToken}`);
        resp = await attempt();
      }
    }
    return resp;
  };
}


