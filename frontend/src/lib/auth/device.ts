import { getApiBaseUrlClient } from '$lib/config.client';

export type DeviceRecord = {
  deviceId: string;
  key: CryptoKey; // non-extractable private key
  pubJwk: JsonWebKey;
  name: string;
};

const DB_NAME = 'forge-auth';
const STORE = 'device';

async function db<T>(mode: IDBTransactionMode, fn: (s: IDBObjectStore) => Promise<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onerror = () => reject(req.error);
    req.onsuccess = () => {
      const tx = req.result.transaction(STORE, mode);
      const store = tx.objectStore(STORE);
      fn(store)
        .then((v) => {
          tx.oncomplete = () => resolve(v);
        })
        .catch(reject);
    };
  });
}

async function putDevice(val: any) {
  return db('readwrite', (s) =>
    new Promise((res, rej) => {
      const r = s.put(val, 'device');
      r.onsuccess = () => res(undefined as any);
      r.onerror = () => rej(r.error);
    })
  );
}

async function getDevice(): Promise<DeviceRecord | null> {
  return db('readonly', (s) =>
    new Promise((res, rej) => {
      const r = s.get('device');
      r.onsuccess = () => res((r.result as any) || null);
      r.onerror = () => rej(r.error);
    })
  );
}

export const deviceKey = {
  async getOrCreate(nameHint = 'This browser'): Promise<DeviceRecord> {
    let rec = await getDevice();
    if (rec?.deviceId && rec?.key) return rec as DeviceRecord;

    const keys = await crypto.subtle.generateKey(
      { name: 'ECDSA', namedCurve: 'P-256' },
      false,
      ['sign']
    );
    const pubJwk = (await crypto.subtle.exportKey('jwk', keys.publicKey)) as JsonWebKey;
    const deviceId = crypto.randomUUID();
    rec = { deviceId, key: keys.privateKey, pubJwk, name: nameHint } as DeviceRecord;
    await putDevice(rec);
    return rec;
  }
};

function b64urlFromBytes(bytes: Uint8Array): string {
  let s = '';
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

export async function buildRegisterProof(key: CryptoKey, deviceId: string, challenge: string, ts: number): Promise<string> {
  const canonical = `DEVICE-REGISTER\n${ts}\n${deviceId}\n${challenge}`;
  const msg = new TextEncoder().encode(canonical);
  // WebCrypto hashes the input with the specified hash internally
  const sig = await crypto.subtle.sign({ name: 'ECDSA', hash: 'SHA-256' }, key, msg);
  // WebCrypto returns DER-encoded ECDSA for ECDSA.sign with namedCurve P-256
  return `${ts}.${b64urlFromBytes(new Uint8Array(sig))}`;
}

export async function buildRefreshProof(key: CryptoKey, deviceId: string, originHost: string, method: string, path: string, ts: number): Promise<string> {
  const canonical = `AUTH-REFRESH\n${ts}\n${deviceId}\n${originHost}\n${method} ${path}`;
  const msg = new TextEncoder().encode(canonical);
  // Need raw r||s. Some browsers return DER, others may return raw r||s.
  const sig = new Uint8Array(
    await crypto.subtle.sign({ name: 'ECDSA', hash: 'SHA-256' }, key, msg)
  );
  if (sig.length === 64 && sig[0] !== 0x30) {
    // Already raw r||s
    return `${ts}.${b64urlFromBytes(sig)}`;
  }
  // Parse DER → r||s
  let i = 0;
  if (sig[i++] !== 0x30) throw new Error('bad DER');
  if (sig[i] & 0x80) i += 1 + (sig[i] & 0x7f); else i += 1; // skip seq len
  if (sig[i++] !== 0x02) throw new Error('bad DER r');
  let rlen = sig[i++];
  let r = sig.slice(i, i + rlen); i += rlen;
  if (sig[i++] !== 0x02) throw new Error('bad DER s');
  let slen = sig[i++];
  let s = sig.slice(i, i + slen);
  // trim leading zeros
  while (r.length > 32 && r[0] === 0) r = r.slice(1);
  while (s.length > 32 && s[0] === 0) s = s.slice(1);
  const rb = new Uint8Array(32); rb.set(r, 32 - r.length);
  const sb = new Uint8Array(32); sb.set(s, 32 - s.length);
  const raw = new Uint8Array(64); raw.set(rb, 0); raw.set(sb, 32);
  return `${ts}.${b64urlFromBytes(raw)}`;
}

export async function registerDevice(init: { device_id: string; challenge: string; alg: string; }, name = 'This browser') {
  const rec = await deviceKey.getOrCreate(name);
  // Use server-provided device_id for registration and persist it for future refreshes
  rec.deviceId = init.device_id;
  await putDevice(rec);
  const ts = Math.floor(Date.now() / 1000);
  const proof = await buildRegisterProof(rec.key, rec.deviceId, init.challenge, ts);
  const API = getApiBaseUrlClient();
  const url = API ? `${API}/auth/devices/register` : `/auth/devices/register`;
  const resp = await fetch(url, {
    method: 'POST',
    credentials: 'include',
    headers: { 'content-type': 'application/json', accept: 'application/json' },
    body: JSON.stringify({
      device_id: rec.deviceId,
      device_name: rec.name,
      public_key_jwk: rec.pubJwk,
      device_proof: proof,
      timestamp: ts
    })
  });
  if (!resp.ok) throw new Error(`device register failed: ${resp.status}`);
  const json = await resp.json();
  // Provide tokens to session cache
  try {
    const { setTokens } = await import('$lib/auth/session');
    if (json?.access_token && json?.expires_in) setTokens(json.access_token, json.expires_in);
  } catch { }
  // Set a lightweight session hint cookie for SSR routing/nav (dev-only convenience)
  try {
    const email = (json?.user?.email as string) || 'device';
    const expires = new Date(Date.now() + 86400 * 1000).toUTCString();
    document.cookie = `cf_session=${encodeURIComponent(email)}; Path=/; Expires=${expires}`;
  } catch { }
  return json;
}


