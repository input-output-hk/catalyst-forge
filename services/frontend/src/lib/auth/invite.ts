/**
 * Invitation token management utilities.
 *
 * This module handles parsing and validation of invitation tokens,
 * which are used to invite new users to the platform.
 *
 * Supports:
 * - Base64URL-encoded JSON tokens
 * - Token expiration validation
 * - Token usage tracking
 * - Demo tokens for testing
 *
 * Moved from src/lib/invite.ts to src/lib/auth/invite.ts
 */

// Type definitions
export type InviteStatus = "valid" | "expired" | "used" | "invalid";

export interface InvitePayload {
  email: string;
  invitedBy?: string;
  expiresAt: string; // ISO 8601 datetime string
  org?: string;
  role?: string;
  status?: "used"; // Marks token as already used
}

export interface ParsedInvite {
  status: InviteStatus;
  email?: string;
  invitedBy?: string;
  expiresAt?: Date;
  org?: string;
  role?: string;
  reason?: string; // Error description for invalid tokens
}

/**
 * Decodes a base64url-encoded string to plain text.
 * Handles URL-safe base64 by converting characters and adding padding.
 * @param input - Base64url encoded string
 * @returns Decoded string, or empty string if decoding fails
 */
function base64UrlDecode(input: string): string {
  try {
    // Convert URL-safe characters to standard base64
    const standardBase64 = input.replace(/-/g, "+").replace(/_/g, "/");

    // Calculate and add required padding
    const paddingNeeded = standardBase64.length % 4;
    const paddedBase64 = paddingNeeded
      ? standardBase64 + "=".repeat(4 - paddingNeeded)
      : standardBase64;

    // Decode to plain text
    return atob(paddedBase64);
  } catch {
    return "";
  }
}

/**
 * Parses and validates an invitation token.
 *
 * Accepts a cryptographically secure invite token (base64url-encoded JSON)
 * and returns a normalized, user-friendly object for the UI.
 *
 * Special demo tokens:
 * - "demo": Valid token expiring in 24 hours
 * - "demo-expired": Expired token from 1 hour ago
 * - "demo-used": Already used token
 *
 * @param token - The invitation token to parse (base64url string or demo token)
 * @returns ParsedInvite object with status and decoded payload
 */
export function parseInviteToken(token: string | undefined | null): ParsedInvite {
  // Handle missing token
  if (!token) {
    return {
      status: "invalid",
      reason: "Missing token",
    };
  }

  // Handle demo token for testing
  if (token === "demo") {
    const twentyFourHoursFromNow = Date.now() + 1000 * 60 * 60 * 24;
    const expirationDate = new Date(twentyFourHoursFromNow);

    return {
      status: "valid",
      email: "you@company.com",
      invitedBy: "admin@company.com",
      expiresAt: expirationDate,
      role: "Developer",
    };
  }

  // Handle expired demo token
  if (token === "demo-expired") {
    const oneHourAgo = Date.now() - 1000 * 60 * 60;
    const expiredDate = new Date(oneHourAgo);

    return {
      status: "expired",
      email: "you@company.com",
      invitedBy: "admin@company.com",
      expiresAt: expiredDate,
      role: "Developer",
    };
  }

  // Handle used demo token
  if (token === "demo-used") {
    const oneHourAgo = Date.now() - 1000 * 60 * 60;
    const usedDate = new Date(oneHourAgo);

    return {
      status: "used",
      email: "you@company.com",
      invitedBy: "admin@company.com",
      expiresAt: usedDate,
      role: "Developer",
    };
  }
  // Parse real token
  try {
    // Step 1: Decode the base64url token
    const decodedJson = base64UrlDecode(token);

    if (!decodedJson) {
      return {
        status: "invalid",
        reason: "Malformed token",
      };
    }

    // Step 2: Parse the JSON payload
    const payload = JSON.parse(decodedJson) as InvitePayload;

    // Step 3: Validate required fields
    const hasRequiredFields = payload?.email && payload?.expiresAt;

    if (!hasRequiredFields) {
      return {
        status: "invalid",
        reason: "Incomplete payload",
      };
    }

    // Step 4: Parse expiration date
    const expirationDate = new Date(payload.expiresAt);
    const isValidDate = !Number.isNaN(expirationDate.getTime());

    if (!isValidDate) {
      return {
        status: "invalid",
        reason: "Invalid expiry",
      };
    }

    // Step 5: Build the parsed invite object
    const parsedInvite = {
      email: payload.email,
      invitedBy: payload.invitedBy,
      expiresAt: expirationDate,
      org: payload.org,
      role: payload.role,
    };

    // Step 6: Check if token is marked as used
    if (payload.status === "used") {
      return {
        status: "used" as const,
        ...parsedInvite,
      };
    }

    // Step 7: Check if token has expired
    const currentTime = Date.now();
    const hasExpired = expirationDate.getTime() <= currentTime;

    if (hasExpired) {
      return {
        status: "expired" as const,
        ...parsedInvite,
      };
    }

    // Step 8: Token is valid
    return {
      status: "valid" as const,
      ...parsedInvite,
    };
  } catch {
    return {
      status: "invalid",
      reason: "Unparsable token",
    };
  }
}

/**
 * Formats a date for display in invitation contexts.
 * Uses Intl.DateTimeFormat for locale-aware formatting.
 * @param dt - The date to format
 * @returns Formatted date string, or empty string if date is undefined
 */
export function formatExpiry(dt?: Date): string {
  if (!dt) {
    return "";
  }

  try {
    const formatOptions: Intl.DateTimeFormatOptions = {
      year: "numeric",
      month: "short",
      day: "2-digit",
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
      timeZoneName: "short",
    };

    const formatter = new Intl.DateTimeFormat(undefined, formatOptions);

    return formatter.format(dt);
  } catch {
    // Fallback to basic locale string if Intl fails
    return dt.toLocaleString();
  }
}
