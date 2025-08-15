export type InviteStatus = "valid" | "expired" | "used" | "invalid";

export interface InvitePayload {
  email: string;
  invitedBy?: string;
  expiresAt: string; // ISO string
  org?: string;
  role?: string;
  // optional field that can explicitly mark usage
  status?: "used";
}

export interface ParsedInvite {
  status: InviteStatus;
  email?: string;
  invitedBy?: string;
  expiresAt?: Date;
  org?: string;
  role?: string;
  reason?: string;
}

function base64UrlDecode(input: string): string {
  try {
    // Handle URL-safe base64
    let b64 = input.replace(/-/g, "+").replace(/_/g, "/");
    // Pad with '='
    const pad = b64.length % 4;
    if (pad) b64 += "=".repeat(4 - pad);
    return atob(b64);
  } catch (_) {
    return "";
  }
}

/**
 * parseInviteToken
 * Accepts a cryptographically secure invite token (mocked here as base64url JSON)
 * and returns a normalized, user-friendly object for the UI.
 *
 * Supports a special "demo" token for quick testing.
 */
export function parseInviteToken(token: string | undefined | null): ParsedInvite {
  if (!token) return { status: "invalid", reason: "Missing token" };

  // Demo shortcut for showcasing the flow without crafting a token
  if (token === "demo") {
    const future = new Date(Date.now() + 1000 * 60 * 60 * 24); // +24h
    return {
      status: "valid",
      email: "you@company.com",
      invitedBy: "admin@company.com",
      expiresAt: future,
      role: "Developer",
    };
  }
  if (token === "demo-expired") {
    const past = new Date(Date.now() - 1000 * 60 * 60); // 1h ago
    return {
      status: "expired",
      email: "you@company.com",
      invitedBy: "admin@company.com",
      expiresAt: past,
      role: "Developer",
    };
  }
  if (token === "demo-used") {
    const past = new Date(Date.now() - 1000 * 60 * 60); // 1h ago
    return {
      status: "used",
      email: "you@company.com",
      invitedBy: "admin@company.com",
      expiresAt: past,
      role: "Developer",
    };
  }

  try {
    const json = base64UrlDecode(token);
    if (!json) return { status: "invalid", reason: "Malformed token" };

    const payload = JSON.parse(json) as InvitePayload;
    if (!payload?.email || !payload?.expiresAt) {
      return { status: "invalid", reason: "Incomplete payload" };
    }

    const expires = new Date(payload.expiresAt);
    if (payload.status === "used") {
      return {
        status: "used",
        email: payload.email,
        invitedBy: payload.invitedBy,
        expiresAt: expires,
        org: payload.org,
        role: payload.role,
      };
    }

    if (Number.isNaN(expires.getTime())) {
      return { status: "invalid", reason: "Invalid expiry" };
    }

    if (expires.getTime() <= Date.now()) {
      return {
        status: "expired",
        email: payload.email,
        invitedBy: payload.invitedBy,
        expiresAt: expires,
        org: payload.org,
        role: payload.role,
      };
    }

    return {
      status: "valid",
      email: payload.email,
      invitedBy: payload.invitedBy,
      expiresAt: expires,
      org: payload.org,
      role: payload.role,
    };
  } catch (err) {
    return { status: "invalid", reason: "Unparsable token" };
  }
}

export function formatExpiry(dt?: Date): string {
  if (!dt) return "";
  try {
    return new Intl.DateTimeFormat(undefined, {
      year: "numeric",
      month: "short",
      day: "2-digit",
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
      timeZoneName: "short",
    }).format(dt);
  } catch {
    return dt.toLocaleString();
  }
}
