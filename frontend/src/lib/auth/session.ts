import { getApiBaseUrlClient } from '$lib/config.client';
import { buildRefreshProof, deviceKey } from '$lib/auth/device';

type Tokens = { accessToken: string; exp: number };
let TOKENS: Tokens | null = null;

function now(): number {
  return Math.floor(Date.now() / 1000);
}

function getOriginHost(): string {
  const base = getApiBaseUrlClient();
  if (!base && typeof window !== 'undefined') return window.location.host;
  try {
    return new URL(base).host;
  } catch {
    return '';
  }
}

export function setTokens(accessToken: string, expiresInSeconds: number) {
  TOKENS = { accessToken, exp: now() + expiresInSeconds };
}

export async function ensureAccessToken(): Promise<string | null> {
  if (TOKENS && TOKENS.exp - now() > 60) return TOKENS.accessToken;

  // Build refresh proof with device key and call /auth/refresh
  const rec = await (async () => {
    // Ensure a device exists; if not, we cannot refresh
    try {
      // deviceKey.getOrCreate would create a new deviceId which would not be linked
      // so we read whatever is stored, if any
      // @ts-ignore access internal helper exposed by device module
      const anyDev = await (deviceKey as any).getOrCreate?.();
      return anyDev ?? null;
    } catch {
      return null;
    }
  })();

  if (!rec?.deviceId || !rec?.key) return null;

  const t = Math.floor(Date.now() / 1000);
  const originHost = getOriginHost();
  const proof = await buildRefreshProof(rec.key, rec.deviceId, originHost, 'POST', '/auth/refresh', t);

  const base = getApiBaseUrlClient();
  const url = base ? `${base}/auth/refresh` : '/auth/refresh';
  const resp = await fetch(url, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'X-Device-Id': rec.deviceId,
      'X-Device-Proof': proof
    }
  });
  if (!resp.ok) return null;
  const json = await resp.json();
  TOKENS = { accessToken: json.access_token, exp: now() + json.expires_in };
  return TOKENS.accessToken;
}

export function withAuth(doFetch = fetch) {
  return async (input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> => {
    // If input is a string path and no explicit base is configured, keep it relative so dev proxy applies
    // When a full API base is configured, rewrite string inputs to that base
    const base = getApiBaseUrlClient();
    let url: RequestInfo | URL = input;
    if (typeof input === 'string') {
      if (base) {
        url = input.startsWith('http') ? input : `${base}${input.startsWith('/') ? '' : '/'}${input}`;
      }
    }
    const at = await ensureAccessToken();
    if (!at) return new Response(null as any, { status: 401 });
    const headers = new Headers(init.headers || {});
    headers.set('Authorization', `Bearer ${at}`);
    let res = await doFetch(url, { ...init, headers, credentials: 'include' });
    if (res.status === 401) {
      const at2 = await ensureAccessToken();
      if (!at2) return res;
      headers.set('Authorization', `Bearer ${at2}`);
      res = await doFetch(url, { ...init, headers, credentials: 'include' });
    }
    return res;
  };
}

export async function logout(): Promise<void> {
  // Best-effort device-bound logout
  try {
    const rec = await (deviceKey as any).getOrCreate?.();
    if (!rec?.deviceId || !rec?.key) return;
    const t = Math.floor(Date.now() / 1000);
    const originHost = getOriginHost();
    const proof = await buildRefreshProof(rec.key, rec.deviceId, originHost, 'POST', '/auth/logout', t);
    const base = getApiBaseUrlClient();
    const url = base ? `${base}/auth/logout` : '/auth/logout';
    await fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'X-Device-Id': rec.deviceId,
        'X-Device-Proof': proof
      }
    });
  } finally {
    TOKENS = null;
    // Clear the lightweight session hint cookie
    try {
      document.cookie = 'cf_session=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT';
    } catch { }
  }
}


