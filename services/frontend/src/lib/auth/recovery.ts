/**
 * Recovery key management utilities.
 *
 * This module provides functions for generating, validating, and managing
 * recovery keys used for account recovery. Uses an unambiguous Base32 alphabet
 * (uppercase A-Z without I, L, O; digits 2-7) to prevent confusion.
 *
 * IMPORTANT: Frontend-only demo generation. Do not use as-is for production secrets.
 * Real recovery keys should be generated server-side for security.
 *
 * Moved from src/lib/recovery.ts to src/lib/auth/recovery.ts
 */

import { forge } from "@/lib/client";
import type { components } from "forge-client";
import { readResponseError } from "@/lib/api";

// Type definitions
type RecoveryGenerateResponse = components["schemas"]["auth.RecoveryGenerateResponse"];

// Unambiguous Base32 alphabet: excludes I, L, O (confusable letters) and 0, 1 (confusable digits)
export const RECOVERY_ALPHABET = "ABCDEFGHJKMNPQRSTUVWXYZ234567";

/**
 * Generates a single random character from the recovery alphabet.
 * Uses crypto.getRandomValues for cryptographic randomness.
 * @returns A single character from RECOVERY_ALPHABET
 */
function randomChar(): string {
  const randomBuffer = new Uint32Array(1);
  crypto.getRandomValues(randomBuffer);

  const randomIndex = randomBuffer[0] % RECOVERY_ALPHABET.length;

  return RECOVERY_ALPHABET[randomIndex];
}

/**
 * Normalizes user input to match recovery key format.
 * Removes formatting, converts to uppercase, and strips invalid characters.
 * @param value - Raw user input
 * @returns Normalized string containing only valid recovery key characters
 */
export function normalizeRecoveryKeyInput(value: string): string {
  // Step 1: Convert to uppercase
  const uppercased = value.toUpperCase();

  // Step 2: Remove all non-alphabet and non-digit characters
  const alphanumericOnly = uppercased.replace(/[^A-Z2-7]/g, "");

  // Step 3: Remove ambiguous letters (I, L, O)
  const unambiguousOnly = alphanumericOnly.replace(/[ILO]/g, "");

  return unambiguousOnly;
}

/**
 * Formats a recovery key with hyphens for readability.
 * Groups characters into chunks (default: groups of 5).
 * @param value - The recovery key to format
 * @param groupSize - Number of characters per group (default: 5)
 * @returns Formatted key with hyphen separators (e.g., "ABCDE-FGHJK-MNPQR")
 */
export function formatRecoveryKey(value: string, groupSize = 5): string {
  const normalizedKey = normalizeRecoveryKeyInput(value);
  const groups: string[] = [];

  // Split into groups of specified size
  for (let i = 0; i < normalizedKey.length; i += groupSize) {
    const group = normalizedKey.slice(i, i + groupSize);
    groups.push(group);
  }

  return groups.join("-");
}

/**
 * Validates if a string is a valid recovery key.
 * Checks both length and character set.
 * @param value - The key to validate
 * @param validLengths - Acceptable key lengths (default: [20, 25, 26])
 * @returns True if the key is valid
 */
export function isValidRecoveryKey(value: string, validLengths: number[] = [20, 25, 26]): boolean {
  const normalizedKey = normalizeRecoveryKeyInput(value);

  // Check length validity
  const hasValidLength = validLengths.includes(normalizedKey.length);

  if (!hasValidLength) {
    return false;
  }

  // Check character validity
  const hasOnlyValidChars = /^[A-Z2-7]+$/.test(normalizedKey);

  return hasOnlyValidChars;
}

/**
 * Generates a raw recovery key of specified length.
 * @param len - The desired length of the key
 * @returns Unformatted recovery key string
 */
function makeKeyRaw(len: number): string {
  let keyString = "";

  for (let i = 0; i < len; i++) {
    keyString += randomChar();
  }

  return keyString;
}

/**
 * Generates multiple formatted recovery keys.
 * Default: 8 keys, each with 5 groups of 5 characters (25 chars total).
 * @param count - Number of keys to generate (default: 8)
 * @param groups - Number of groups per key (default: 5)
 * @param groupLen - Characters per group (default: 5)
 * @returns Array of formatted recovery key strings
 */
export function generateRecoveryKeys(count = 8, groups = 5, groupLen = 5): string[] {
  // Ensure positive values
  const safeGroups = Math.max(1, groups);
  const safeGroupLen = Math.max(1, groupLen);
  const totalLength = safeGroups * safeGroupLen;

  // Generate array of formatted keys
  const keys = Array.from({ length: count }, () => {
    const rawKey = makeKeyRaw(totalLength);
    return formatRecoveryKey(rawKey, safeGroupLen);
  });

  return keys;
}

/**
 * Converts recovery keys to a formatted text string.
 * Useful for display or saving to a file.
 * @param keys - Array of recovery key strings
 * @param header - Header text for the output (default: "Recovery Keys")
 * @returns Formatted text with numbered keys
 */
export function keysToText(keys: string[], header = "Recovery Keys"): string {
  const headerLine = header;

  const keyLines = keys.map((key, index) => {
    const keyNumber = index + 1;
    return `Key ${keyNumber}: ${key}`;
  });

  return `${headerLine}\n\n${keyLines.join("\n")}\n`;
}

/**
 * Downloads recovery codes as a text file.
 * @param filename - Name for the downloaded file
 * @param codes - Array of recovery codes to save
 */
export function downloadRecoveryCodes(filename: string, codes: string[]): void {
  // Create text content
  const textContent = codes.join("\n");

  // Create blob with UTF-8 encoding
  const blob = new Blob([textContent], { type: "text/plain;charset=utf-8" });

  // Create download URL
  const downloadUrl = URL.createObjectURL(blob);

  // Create and trigger download link
  const downloadLink = document.createElement("a");
  downloadLink.href = downloadUrl;
  downloadLink.download = filename;
  downloadLink.click();

  // Clean up object URL
  URL.revokeObjectURL(downloadUrl);
}

/**
 * Copies recovery codes to the clipboard.
 * Fails silently if clipboard API is unavailable.
 * @param codes - Array of recovery codes to copy
 */
export async function copyRecoveryCodesToClipboard(codes: string[]): Promise<void> {
  const textToCopy = codes.join("\n");

  try {
    await navigator.clipboard.writeText(textToCopy);
  } catch {
    // Clipboard may be unavailable (HTTP context, permissions)
    // Fail silently - callers can handle user feedback
  }
}

/**
 * Regenerates recovery codes via API.
 * Falls back to client-side generation if API fails.
 * @returns Array of new recovery codes
 * @throws Error if API returns an error message
 */
export async function regenerateRecoveryCodes(): Promise<string[]> {
  const response = await forge.raw.POST("/api/v1/auth/recovery/codes/generate");

  if (!response.response.ok) {
    // Try to get error message from server
    const errorMessage = await readResponseError(response, "");

    if (errorMessage) {
      throw new Error(errorMessage);
    }

    // Fallback to client-side generation
    return generateRecoveryKeys(8);
  }

  // Extract codes from response
  const responseData = response.data as RecoveryGenerateResponse;
  const serverCodes = responseData?.codes;
  const hasValidCodes = Array.isArray(serverCodes);

  // Use server codes if valid, otherwise generate client-side
  return hasValidCodes ? (serverCodes as string[]) : generateRecoveryKeys(8);
}

/**
 * Backward compatibility alias for regenerateRecoveryCodes.
 * Swallows errors and returns empty array on failure.
 * @returns Array of recovery codes, or empty array if generation fails
 * @deprecated Use regenerateRecoveryCodes() directly
 */
export async function requestRecoveryCodes(): Promise<string[]> {
  try {
    const codes = await regenerateRecoveryCodes();
    return codes;
  } catch {
    return [];
  }
}

/**
 * Requests recovery codes and starts the recovery gate UI.
 * @param startRecoveryGate - Function to display recovery gate UI
 * @param returnTo - Optional URL to return to after recovery
 * @deprecated Use regenerateAndStartGate() instead
 */
export async function requestCodesAndStartGate(
  startRecoveryGate: (keys: string[], returnTo?: string) => void,
  returnTo?: string
): Promise<void> {
  const codes = await requestRecoveryCodes().catch(() => []);

  startRecoveryGate(codes, returnTo);
}

/**
 * Regenerates recovery codes and starts the recovery gate UI.
 * @param startRecoveryGate - Function to display recovery gate UI
 * @param returnTo - Optional URL to return to after recovery
 */
export async function regenerateAndStartGate(
  startRecoveryGate: (keys: string[], returnTo?: string) => void,
  returnTo?: string
): Promise<void> {
  const codes = await regenerateRecoveryCodes();

  startRecoveryGate(codes, returnTo);
}
