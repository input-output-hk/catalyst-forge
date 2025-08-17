/**
 * WebAuthn utility functions for passwordless authentication.
 *
 * This module provides helper functions for WebAuthn operations including:
 * - Authentication (login) flows
 * - Registration (credential creation) flows
 * - Binary data encoding/decoding (base64url <-> ArrayBuffer)
 * - Server response normalization
 *
 * Handles various server response formats and provides fallbacks for browsers
 * that don't support the latest WebAuthn JSON parsing APIs.
 */

import { forgeFetch, forge } from "./client";

// Type definitions for better code clarity
type PublicKeyCredentialWithHelpers = PublicKeyCredential & {
  toJSON?: () => unknown;
};

interface PublicKeyCredentialConstructorWithParsers {
  parseRequestOptionsFromJSON?: (
    options: Record<string, unknown>
  ) => PublicKeyCredentialRequestOptions;
  parseCreationOptionsFromJSON?: (
    options: Record<string, unknown>
  ) => PublicKeyCredentialCreationOptions;
  isConditionalMediationAvailable?: () => Promise<boolean>;
}

interface BeginLoginResponse {
  publicKey?: unknown;
  session_key?: string;
}

interface CompleteLoginResponse {
  access_token?: string;
}

interface CredentialIdVariants {
  id?: unknown;
  credentialId?: unknown;
  credential_id?: unknown;
}

/**
 * Converts a base64 string to a Uint8Array.
 * @param b64 - Base64 encoded string
 * @returns Uint8Array representation of the base64 string
 */
function b64ToBytes(b64: string): Uint8Array {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);

  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }

  return bytes;
}

/**
 * Converts various binary representations to a Uint8Array.
 * Handles ArrayBuffer, ArrayBufferView, base64 strings, base64url strings, and number arrays.
 * @param value - The value to convert (can be ArrayBuffer, string, array, etc.)
 * @returns Uint8Array representation of the input
 * @throws Error if the value type is not supported
 */
function toBytesUnknown(value: unknown): Uint8Array {
  if (!value) {
    return new Uint8Array();
  }

  if (value instanceof ArrayBuffer) {
    return new Uint8Array(value);
  }

  if (ArrayBuffer.isView(value)) {
    const bufferView = value as ArrayBufferView;
    return new Uint8Array(bufferView.buffer);
  }

  if (typeof value === "string") {
    // Try URL-safe base64 first, then standard base64
    try {
      const paddingNeeded = (value.length + 3) % 4;
      const padding = "===".slice(paddingNeeded);
      const standardBase64 = value.replace(/-/g, "+").replace(/_/g, "/");
      const paddedBase64 = standardBase64 + padding;

      return b64ToBytes(paddedBase64);
    } catch {
      return b64ToBytes(value);
    }
  }

  if (Array.isArray(value)) {
    return new Uint8Array(value as number[]);
  }

  throw new Error("Unsupported binary value");
}

/**
 * Converts an ArrayBuffer to a base64url encoded string.
 * @param buf - ArrayBuffer to encode
 * @returns Base64url encoded string (URL-safe, no padding)
 */
function arrayBufferToB64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let binary = "";

  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }

  const base64 = btoa(binary);
  const base64url = base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");

  return base64url;
}

/**
 * Checks if the browser supports native WebAuthn JSON parsing helpers.
 * These helpers were added in newer browser versions to simplify WebAuthn operations.
 * @returns True if native parsing methods are available, false otherwise
 */
function hasParseHelpers(): boolean {
  if (typeof PublicKeyCredential === "undefined") {
    return false;
  }

  const PKC = PublicKeyCredential as unknown as PublicKeyCredentialConstructorWithParsers;

  const hasRequestParser = typeof PKC.parseRequestOptionsFromJSON === "function";
  const hasCreationParser = typeof PKC.parseCreationOptionsFromJSON === "function";

  return hasRequestParser && hasCreationParser;
}

/**
 * Converts server-provided options to PublicKeyCredentialRequestOptions.
 * Uses native browser parsing if available, otherwise manually normalizes the data.
 * @param opts - Raw options object from server
 * @returns Properly formatted PublicKeyCredentialRequestOptions for WebAuthn API
 * @throws Error if challenge is missing or invalid
 */
function toPublicKeyRequestOptions(
  opts: Record<string, unknown>
): PublicKeyCredentialRequestOptions {
  // Use native parser if available
  if (hasParseHelpers()) {
    const PKC = PublicKeyCredential as unknown as PublicKeyCredentialConstructorWithParsers;
    return PKC.parseRequestOptionsFromJSON!(opts);
  }

  // Manual parsing fallback
  const normalizedOptions: Record<string, unknown> = { ...opts };

  // Normalize challenge
  normalizedOptions.challenge = toBytesUnknown(normalizedOptions.challenge);

  const challengeBytes = normalizedOptions.challenge as Uint8Array;
  if (!(challengeBytes instanceof Uint8Array) || challengeBytes.byteLength === 0) {
    throw new Error("Server did not provide a valid publicKey.challenge");
  }

  // Normalize allowCredentials
  if (Array.isArray(normalizedOptions.allowCredentials)) {
    const credentials = normalizedOptions.allowCredentials as Array<Record<string, unknown>>;

    normalizedOptions.allowCredentials = credentials.map((credential) => {
      const credentialWithIds = credential as CredentialIdVariants;

      // Try different possible ID field names
      const credentialId =
        credentialWithIds.id ?? credentialWithIds.credentialId ?? credentialWithIds.credential_id;

      const descriptor: PublicKeyCredentialDescriptor = {
        type: "public-key",
        id: toBytesUnknown(credentialId),
        transports: (credential as { transports?: AuthenticatorTransport[] }).transports,
      };

      return descriptor;
    });
  }

  // Parse timeout if it's a string
  if (typeof normalizedOptions.timeout === "string") {
    const timeoutValue = parseInt(normalizedOptions.timeout as string, 10);
    normalizedOptions.timeout = timeoutValue || undefined;
  }

  return normalizedOptions as unknown as PublicKeyCredentialRequestOptions;
}

/**
 * Maps WebAuthn DOM exceptions to user-friendly error messages.
 * @param e - The error thrown by WebAuthn API
 * @returns Error with a user-friendly message
 */
function mapDomError(e: unknown): Error {
  if (!e || typeof e !== "object" || !("name" in e)) {
    return new Error(String(e));
  }

  const errorWithName = e as { name?: string; message?: string };
  const errorName = errorWithName.name;

  switch (errorName) {
    case "NotAllowedError":
      return new Error("Authentication was cancelled or timed out.");

    case "InvalidStateError":
      return new Error("Credential is already registered on this origin.");

    case "SecurityError":
      return new Error("Security policy blocked WebAuthn on this page.");

    case "NotSupportedError":
      return new Error("This authenticator or algorithm isn't supported.");

    default:
      return new Error(errorWithName.message || String(e));
  }
}

/**
 * Performs WebAuthn authentication login flow.
 * Handles the complete authentication process from credential request to token storage.
 * @param opts - Optional configuration
 * @param opts.conditionalUi - Enable conditional UI (autofill) if supported
 * @param opts.signal - AbortSignal to cancel the authentication
 * @throws Error if authentication fails at any step
 */
export async function loginWithWebAuthn(opts?: {
  conditionalUi?: boolean;
  signal?: AbortSignal;
}): Promise<void> {
  // Step 1: Begin authentication flow
  // Create a fresh Request to avoid reusing a consumed Request object across retries
  const beginResponse = await forgeFetch("/api/v1/auth/login/begin", {
    method: "POST",
    cache: "no-store",
  });

  if (!beginResponse.ok) {
    throw new Error(`begin ${beginResponse.status}`);
  }

  const beginData = (await beginResponse.json()) as BeginLoginResponse;

  // Step 2: Extract and normalize publicKey options
  // Some servers wrap the options as { publicKey: { publicKey: {...} } }
  const rawPublicKeyOptions = extractPublicKeyOptions(beginData);
  const publicKeyOptions = toPublicKeyRequestOptions(rawPublicKeyOptions);

  // Step 3: Configure credential request options
  const credentialRequestOptions = await buildCredentialRequestOptions(
    publicKeyOptions,
    opts?.conditionalUi
  );

  // Step 4: Get credential from authenticator
  let credential: PublicKeyCredential | null = null;

  try {
    const getOptions = {
      ...credentialRequestOptions,
      signal: opts?.signal,
    } as CredentialRequestOptions;

    credential = (await navigator.credentials.get(getOptions)) as PublicKeyCredential | null;
  } catch (e) {
    throw mapDomError(e);
  }

  if (!credential) {
    throw new Error("No credential returned");
  }

  // Step 5: Prepare completion payload
  const completionPayload = buildCompletionPayload(credential, beginData.session_key);

  // Step 6: Complete authentication
  const completeResponse = await forgeFetch("/api/v1/auth/login/complete", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(completionPayload),
    cache: "no-store",
  });

  if (!completeResponse.ok) {
    throw new Error(`complete ${completeResponse.status}`);
  }

  // Step 7: Store access token if provided
  await storeAccessToken(completeResponse);
}

/**
 * Extracts publicKey options from various server response formats.
 * Handles nested structures like { publicKey: { publicKey: {...} } }.
 * @param beginData - Response from login/begin endpoint
 * @returns Normalized publicKey options object
 */
function extractPublicKeyOptions(beginData: BeginLoginResponse): Record<string, unknown> {
  const publicKeyField = beginData.publicKey;

  if (!publicKeyField) {
    return {};
  }

  // Check for nested publicKey.publicKey structure
  const publicKeyWrapper = publicKeyField as { publicKey?: unknown };

  if (publicKeyWrapper.publicKey && typeof publicKeyWrapper.publicKey === "object") {
    // Handle double-nested publicKey.publicKey.publicKey
    const innerWrapper = publicKeyWrapper.publicKey as { publicKey?: unknown };

    if (innerWrapper.publicKey && typeof innerWrapper.publicKey === "object") {
      return innerWrapper.publicKey as Record<string, unknown>;
    }

    return publicKeyWrapper.publicKey as Record<string, unknown>;
  }

  return publicKeyField as Record<string, unknown>;
}

/**
 * Builds credential request options with optional conditional UI support.
 * Conditional UI allows credentials to appear in autofill suggestions.
 * @param publicKey - The public key credential request options
 * @param conditionalUi - Whether to enable conditional UI if available
 * @returns Configured credential request options
 */
async function buildCredentialRequestOptions(
  publicKey: PublicKeyCredentialRequestOptions,
  conditionalUi?: boolean
): Promise<CredentialRequestOptions> {
  type ExtendedCredentialRequestOptions = CredentialRequestOptions & { mediation?: "conditional" };
  const requestOptions: ExtendedCredentialRequestOptions = { publicKey };

  // Check and enable conditional UI if available
  if (conditionalUi) {
    try {
      const PKC = PublicKeyCredential as unknown as PublicKeyCredentialConstructorWithParsers;

      if (typeof PKC.isConditionalMediationAvailable === "function") {
        const isAvailable = await PKC.isConditionalMediationAvailable();

        if (isAvailable) {
          requestOptions.mediation = "conditional";
        }
      }
    } catch {
      // Conditional UI not supported, continue without it
    }
  }

  return requestOptions;
}

/**
 * Builds the completion payload to send to the server after credential creation.
 * Uses native toJSON if available, otherwise manually serializes the credential.
 * @param credential - The public key credential from WebAuthn API
 * @param sessionKey - Session key from the begin response
 * @returns Serialized payload for the complete endpoint
 */
function buildCompletionPayload(
  credential: PublicKeyCredential,
  sessionKey?: string
): Record<string, unknown> {
  const credentialWithHelpers = credential as PublicKeyCredentialWithHelpers;

  // Use native toJSON if available
  if (typeof credentialWithHelpers.toJSON === "function") {
    return {
      session_key: sessionKey || "",
      credential: credentialWithHelpers.toJSON(),
    };
  }

  // Manual serialization fallback
  const assertionResponse = credential.response as AuthenticatorAssertionResponse;

  const serializedCredential = {
    id: credential.id,
    type: credential.type,
    rawId: arrayBufferToB64url(credential.rawId),
    response: {
      clientDataJSON: arrayBufferToB64url(assertionResponse.clientDataJSON),
      authenticatorData: arrayBufferToB64url(assertionResponse.authenticatorData),
      signature: arrayBufferToB64url(assertionResponse.signature),
      userHandle: assertionResponse.userHandle
        ? arrayBufferToB64url(assertionResponse.userHandle as ArrayBuffer)
        : undefined,
    },
    clientExtensionResults: credentialWithHelpers.getClientExtensionResults?.() || {},
  };

  return {
    session_key: sessionKey || "",
    credential: serializedCredential,
  };
}

/**
 * Extracts and stores the access token from the authentication response.
 * @param response - Response from the complete endpoint
 */
async function storeAccessToken(response: Response): Promise<void> {
  try {
    const responseData = (await response.json()) as CompleteLoginResponse | null;

    if (!responseData || !responseData.access_token) {
      return;
    }

    const accessToken = responseData.access_token;

    if (typeof accessToken === "string" && accessToken.length > 0) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const forgeClient = forge as any;
      forgeClient.setAccessToken?.(accessToken);
    }
  } catch {
    // Some servers may return 204; ignore JSON parse errors
  }
}

/**
 * Converts a base64url encoded string to an ArrayBuffer.
 * Used for decoding WebAuthn challenge and credential IDs.
 * @param value - Base64url encoded string
 * @returns ArrayBuffer containing the decoded bytes
 */
export function b64urlToBuf(value: string): ArrayBuffer {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const paddingNeeded = base64.length % 4 ? 4 - (base64.length % 4) : 0;
  const paddedBase64 = base64 + "=".repeat(paddingNeeded);

  const binaryString =
    typeof atob === "function"
      ? atob(paddedBase64)
      : Buffer.from(paddedBase64, "base64").toString("binary");

  const bytes = new Uint8Array(binaryString.length);

  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }

  return bytes.buffer;
}

/**
 * Decodes server-provided creation options for WebAuthn credential registration.
 * Converts base64url encoded fields to ArrayBuffers as required by WebAuthn API.
 * @param options - Raw creation options from server
 * @returns Properly formatted PublicKeyCredentialCreationOptions
 */
export function decodeCreationOptions(
  options: Record<string, unknown>
): PublicKeyCredentialCreationOptions {
  const decodedOptions: Record<string, unknown> = { ...options };

  // Normalize challenge
  if (typeof decodedOptions.challenge === "string") {
    decodedOptions.challenge = b64urlToBuf(decodedOptions.challenge as string);
  }

  // Normalize user.id
  if (decodedOptions.user && typeof (decodedOptions.user as { id?: unknown }).id === "string") {
    const user = decodedOptions.user as { id?: unknown } & Record<string, unknown>;
    decodedOptions.user = {
      ...user,
      id: b64urlToBuf(user.id as string),
    };
  }

  // Normalize excludeCredentials[].id
  if (Array.isArray(decodedOptions.excludeCredentials)) {
    const excludeCredentials = decodedOptions.excludeCredentials as Array<Record<string, unknown>>;

    decodedOptions.excludeCredentials = excludeCredentials.map((credential) => {
      const decodedCredential: Partial<PublicKeyCredentialDescriptor> & Record<string, unknown> = {
        ...credential,
      };

      const credentialId = credential.id as unknown;

      if (typeof credentialId === "string") {
        decodedCredential.id = b64urlToBuf(credentialId);
      }

      if (!decodedCredential.type) {
        decodedCredential.type = "public-key";
      }

      return decodedCredential as unknown as PublicKeyCredentialDescriptor;
    });
  }

  return decodedOptions as unknown as PublicKeyCredentialCreationOptions;
}

/**
 * Converts an ArrayBuffer to a base64url encoded string.
 * Used for encoding WebAuthn responses to send to the server.
 * @param buffer - ArrayBuffer to encode
 * @returns Base64url encoded string (URL-safe, no padding)
 */
export function bufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binaryString = "";

  for (let i = 0; i < bytes.length; i++) {
    binaryString += String.fromCharCode(bytes[i]);
  }

  const base64 =
    typeof btoa === "function"
      ? btoa(binaryString)
      : Buffer.from(binaryString, "binary").toString("base64");

  const base64url = base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");

  return base64url;
}

export interface EncodedAttestation {
  id: string;
  type: string;
  rawId: string;
  response: {
    clientDataJSON: string;
    attestationObject: string;
  };
}

/**
 * Encodes a WebAuthn attestation response for transmission to the server.
 * Converts all ArrayBuffer fields to base64url strings.
 * @param credential - The public key credential from navigator.credentials.create()
 * @returns Encoded attestation object ready for JSON serialization
 */
export function encodeAttestation(credential: PublicKeyCredential): Record<string, never> {
  const attestationResponse = credential.response as AuthenticatorAttestationResponse;
  const encodedRawId = bufferToBase64Url(credential.rawId);

  const encodedCredential: EncodedAttestation = {
    id: encodedRawId,
    type: credential.type,
    rawId: encodedRawId,
    response: {
      clientDataJSON: bufferToBase64Url(attestationResponse.clientDataJSON),
      attestationObject: bufferToBase64Url(attestationResponse.attestationObject),
    },
  };

  return encodedCredential as unknown as Record<string, never>;
}

/**
 * Extracts creation options from various server response formats.
 * Some servers wrap creation options as { publicKey: { ... } } while others return the object directly.
 * This function normalizes both formats.
 * @param publicKeyBlock - The publicKey block from server response
 * @returns Normalized creation options object
 */
export function extractCreationOptionsFromServer(publicKeyBlock: unknown): Record<string, unknown> {
  if (!publicKeyBlock || typeof publicKeyBlock !== "object") {
    return {};
  }

  const maybeWrapper = publicKeyBlock as { publicKey?: unknown };

  if (maybeWrapper.publicKey && typeof maybeWrapper.publicKey === "object") {
    return maybeWrapper.publicKey as Record<string, unknown>;
  }

  return publicKeyBlock as Record<string, unknown>;
}

/**
 * Convenience helper to create a WebAuthn credential from server-provided publicKey block.
 * Handles extraction, decoding, and credential creation in one call.
 * @param publicKeyBlock - The publicKey block from server response (may be wrapped)
 * @returns Promise resolving to the created PublicKeyCredential
 * @throws Error if credential creation fails or is cancelled
 */
export async function createCredentialFromServerPublicKey(
  publicKeyBlock: unknown
): Promise<PublicKeyCredential> {
  const rawOptions = extractCreationOptionsFromServer(publicKeyBlock);
  const decodedOptions = decodeCreationOptions(rawOptions);

  const credential = (await navigator.credentials.create({
    publicKey: decodedOptions,
  })) as PublicKeyCredential;

  return credential;
}
