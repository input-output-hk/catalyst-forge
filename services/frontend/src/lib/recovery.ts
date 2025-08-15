// Recovery key utilities: unambiguous Base32 (uppercase A-Z without I, L, O; digits 2-7)
// NOTE: Frontend-only demo generation. Do not use as-is for production secrets.

export const RECOVERY_ALPHABET = "ABCDEFGHJKMNPQRSTUVWXYZ234567"; // excludes I, L, O and 0,1; uses Base32 digits 2-7

function randomChar(): string {
  const array = new Uint32Array(1);
  crypto.getRandomValues(array);
  return RECOVERY_ALPHABET[array[0] % RECOVERY_ALPHABET.length];
}

export function normalizeRecoveryKeyInput(value: string): string {
  // Uppercase, strip non-allowed chars and dashes/spaces
  const upper = value.toUpperCase();
  const cleaned = upper.replace(/[^A-Z2-7]/g, "");
  // Remove ambiguous letters if any slipped through (I, L, O)
  return cleaned.replace(/[ILO]/g, "");
}

export function formatRecoveryKey(value: string, groupSize = 5): string {
  const normalized = normalizeRecoveryKeyInput(value);
  const parts: string[] = [];
  for (let i = 0; i < normalized.length; i += groupSize) {
    parts.push(normalized.slice(i, i + groupSize));
  }
  return parts.join("-");
}

export function isValidRecoveryKey(value: string, validLengths: number[] = [20, 25, 26]): boolean {
  const normalized = normalizeRecoveryKeyInput(value);
  if (!validLengths.includes(normalized.length)) return false;
  // Ensure only allowed alphabet characters
  return /^[A-Z2-7]+$/.test(normalized);
}

function makeKeyRaw(len: number): string {
  let out = "";
  for (let i = 0; i < len; i++) out += randomChar();
  return out;
}

// Generate formatted recovery keys (default: 5 groups of 5 = 25 chars)
export function generateRecoveryKeys(count = 8, groups = 5, groupLen = 5): string[] {
  const len = Math.max(1, groups) * Math.max(1, groupLen);
  return Array.from({ length: count }, () => formatRecoveryKey(makeKeyRaw(len), groupLen));
}

export function keysToText(keys: string[], header = "Recovery Keys") {
  return `${header}\n\n${keys.map((k, i) => `Key ${i + 1}: ${k}`).join("\n")}\n`;
}
