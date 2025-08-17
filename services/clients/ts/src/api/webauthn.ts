import type { Client } from 'openapi-fetch';
import type { paths } from './schema';
import { ForgeClient } from './client';

// Minimal base64url helpers to avoid extra deps
function b64urlToBuf(value: string): ArrayBuffer {
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/');
  const pad = base64.length % 4 ? 4 - (base64.length % 4) : 0;
  const b64 = base64 + '='.repeat(pad);
  const str = typeof atob === 'function' ? atob(b64) : Buffer.from(b64, 'base64').toString('binary');
  const bytes = new Uint8Array(str.length);
  for (let i = 0; i < str.length; i++) bytes[i] = str.charCodeAt(i);
  return bytes.buffer;
}

function bufToB64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let str = '';
  for (let i = 0; i < bytes.length; i++) str += String.fromCharCode(bytes[i]);
  const b64 = typeof btoa === 'function' ? btoa(str) : Buffer.from(str, 'binary').toString('base64');
  return b64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

// Convert server-provided publicKey options into WebAuthn-friendly structures
function decodeRequestOptions(options: any): PublicKeyCredentialRequestOptions {
  const src: any = options?.publicKey ?? options; // accept nested { publicKey }
  const out: any = { ...src };
  if (Array.isArray(out.allowCredentials)) {
    out.allowCredentials = out.allowCredentials.map((c: any) => ({
      ...c,
      id: typeof c.id === 'string' ? b64urlToBuf(c.id) : c.id,
    }));
  }
  if (typeof out.challenge === 'string') out.challenge = b64urlToBuf(out.challenge);
  return out;
}

function decodeCreationOptions(options: any): PublicKeyCredentialCreationOptions {
  const src: any = options?.publicKey ?? options; // accept nested { publicKey }
  const out: any = { ...src };
  if (typeof out.challenge === 'string') out.challenge = b64urlToBuf(out.challenge);
  if (out.user && typeof out.user.id === 'string') out.user = { ...out.user, id: b64urlToBuf(out.user.id) };
  if (Array.isArray(out.excludeCredentials)) {
    out.excludeCredentials = out.excludeCredentials.map((c: any) => ({
      ...c,
      id: typeof c.id === 'string' ? b64urlToBuf(c.id) : c.id,
    }));
  }
  return out;
}

function encodeAssertion(cred: PublicKeyCredential): any {
  const assertion = cred as PublicKeyCredential & { response: AuthenticatorAssertionResponse };
  return {
    id: cred.id,
    type: cred.type,
    rawId: bufToB64url(cred.rawId),
    response: {
      clientDataJSON: bufToB64url(assertion.response.clientDataJSON),
      authenticatorData: bufToB64url(assertion.response.authenticatorData),
      signature: bufToB64url(assertion.response.signature),
      userHandle: assertion.response.userHandle ? bufToB64url(assertion.response.userHandle) : undefined,
    },
  };
}

function encodeAttestation(cred: PublicKeyCredential): any {
  const attestation = cred as PublicKeyCredential & { response: AuthenticatorAttestationResponse };
  return {
    id: cred.id,
    type: cred.type,
    rawId: bufToB64url(cred.rawId),
    response: {
      clientDataJSON: bufToB64url(attestation.response.clientDataJSON),
      attestationObject: bufToB64url(attestation.response.attestationObject),
      transports: (typeof (attestation.response as any).getTransports === 'function'
        ? (attestation.response as any).getTransports()
        : undefined) as string[] | undefined,
    },
  };
}

export async function login(client: ForgeClient): Promise<void> {
  if (typeof navigator === 'undefined' || !navigator.credentials) throw new Error('WebAuthn not available');
  const raw: any = client.raw as unknown;
  const begin = await raw.POST('/api/v1/auth/login/begin');
  if (!begin.response.ok) throw new Error('login begin failed');
  const { publicKey, session_key } = (begin.data as any) || {};
  const cred = (await navigator.credentials.get({ publicKey: decodeRequestOptions(publicKey) })) as PublicKeyCredential;
  const complete = await raw.POST('/api/v1/auth/login/complete', {
    body: { session_key, credential: encodeAssertion(cred) } as any,
  });
  if (!complete.response.ok) throw new Error('login complete failed');
}

export async function registerCredential(client: ForgeClient, deviceName: string): Promise<void> {
  if (typeof navigator === 'undefined' || !navigator.credentials) throw new Error('WebAuthn not available');
  const raw = client.raw as Client<paths>;
  const begin = await raw.POST('/api/v1/auth/credentials/add/begin', { body: { device_name: deviceName } as any });
  if (!begin.response.ok) throw new Error('register begin failed');
  const { publicKey, session_key } = (begin.data as any) || {};
  const cred = (await navigator.credentials.create({ publicKey: decodeCreationOptions(publicKey) })) as PublicKeyCredential;
  const complete = await raw.POST('/api/v1/auth/credentials/add/complete', {
    body: { session_key, credential: encodeAttestation(cred) } as any,
  });
  if (!complete.response.ok) throw new Error('register complete failed');
}

export async function stepUp(client: ForgeClient): Promise<void> {
  if (typeof navigator === 'undefined' || !navigator.credentials) throw new Error('WebAuthn not available');
  const raw = client.raw as Client<paths>;
  const begin = await raw.POST('/api/v1/auth/step-up/begin');
  if (!begin.response.ok) throw new Error('step-up begin failed');
  const { publicKey, session_key } = (begin.data as any) || {};
  const cred = (await navigator.credentials.get({ publicKey: decodeRequestOptions(publicKey) })) as PublicKeyCredential;
  const complete = await raw.POST('/api/v1/auth/step-up/complete', {
    body: { session_key, credential: encodeAssertion(cred) } as any,
  });
  if (!complete.response.ok) throw new Error('step-up complete failed');
}


