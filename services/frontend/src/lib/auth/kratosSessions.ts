/**
 * Minimal helpers for listing and revoking the current user's Kratos sessions
 * via the public API (cookie-auth in browser).
 */

export interface KratosSession {
  id: string;
  active?: boolean;
  authenticated_at?: string;
  issued_at?: string;
  expires_at?: string;
  ip_address?: string;
  user_agent?: string;
}

export interface KratosWhoAmI {
  id?: string; // session id
}

const basePath = "/.ory/kratos/public";

export async function listMySessions(params?: { active?: boolean }): Promise<KratosSession[]> {
  const search = new URLSearchParams();
  if (typeof params?.active === "boolean") search.set("active", String(params.active));
  const resp = await fetch(`${basePath}/sessions?${search.toString()}`, { credentials: "include" });
  if (!resp.ok) throw new Error(`Failed to list sessions (${resp.status})`);
  const data = (await resp.json()) as unknown;
  return Array.isArray(data) ? (data as KratosSession[]) : [];
}

export async function whoAmI(): Promise<KratosWhoAmI | null> {
  const resp = await fetch(`${basePath}/sessions/whoami`, { credentials: "include" });
  if (!resp.ok) return null;
  const data = (await resp.json()) as unknown as KratosWhoAmI;
  return data ?? null;
}

export async function deleteMySession(sessionId: string): Promise<void> {
  const resp = await fetch(`${basePath}/sessions/${encodeURIComponent(sessionId)}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!resp.ok) throw new Error(`Failed to delete session (${resp.status})`);
}

/** Revoke all other sessions for the current user (keeps current session). */
export async function disableMyOtherSessions(): Promise<void> {
  const resp = await fetch(`${basePath}/sessions`, { method: "DELETE", credentials: "include" });
  if (!resp.ok) throw new Error(`Failed to revoke other sessions (${resp.status})`);
}

/** Revoke other sessions, then start browser logout for the current session. */
export async function logoutEverywhere(): Promise<void> {
  try {
    await disableMyOtherSessions();
  } catch {
    // ignore and still try to log out current session
  }
  const resp = await fetch(`${basePath}/self-service/logout/browser`, {
    credentials: "include",
  });
  if (resp.ok) {
    const json = (await resp.json()) as { logout_url?: string };
    if (json.logout_url) {
      window.location.href = json.logout_url;
      return;
    }
  }
  // Fallback: reload to clear any local state
  window.location.replace("/welcome");
}

