import { forge } from "@/lib/client";
import { readResponseError } from "@/lib/api";
import type { components } from "forge-client";

export type AdminUsersListResponse = components["schemas"]["auth.AdminUsersListResponse"];

export async function adminListUsers(params: {
  q?: string;
  role?: string;
  limit?: number;
  offset?: number;
}): Promise<{ data: AdminUsersListResponse; total?: number }> {
  const res = await forge.raw.GET("/api/v1/admin/users", {
    params: {
      query: {
        q: params.q,
        role: params.role,
        limit: params.limit,
        offset: params.offset,
      },
    },
  });
  if (!res.response.ok) {
    const msg = await readResponseError(res, "Failed to list users");
    throw new Error(msg);
  }
  const totalHdr = res.response.headers.get("X-Total-Count");
  return {
    data: res.data as AdminUsersListResponse,
    total: totalHdr ? parseInt(totalHdr, 10) : undefined,
  };
}

export async function adminDeleteUser(id: string): Promise<void> {
  const res = await forge.raw.DELETE("/api/v1/admin/users/{id}", { params: { path: { id } } });
  if (res.response.status !== 204) {
    const msg = await readResponseError(res, "Failed to delete user");
    throw new Error(msg);
  }
}

export async function adminUpdateUser(id: string, body: Record<string, never>): Promise<void> {
  const res = await forge.raw.PATCH("/api/v1/admin/users/{id}", { params: { path: { id } }, body });
  if (res.response.status !== 204) {
    const msg = await readResponseError(res, "Failed to update user");
    throw new Error(msg);
  }
}

export type AdminUserCredentialsResponse = {
  credentials: Array<{
    id: string;
    device_name: string;
    sign_count: number;
    last_used_at?: string;
  }>;
};
export async function adminListUserCredentials(id: string): Promise<AdminUserCredentialsResponse> {
  const res = await forge.raw.GET("/api/v1/admin/users/{id}/credentials", {
    params: { path: { id } },
  });
  if (!res.response.ok) {
    const msg = await readResponseError(res, "Failed to load credentials");
    throw new Error(msg);
  }
  return res.data as unknown as AdminUserCredentialsResponse;
}

export type AdminAuditListResponse = components["schemas"]["auth.AuditListResponse"];
export async function adminListAudit(params: {
  user_id?: string;
  limit?: number;
}): Promise<AdminAuditListResponse> {
  const res = await forge.raw.GET("/api/v1/admin/audit", {
    params: { query: { user_id: params.user_id, limit: params.limit } },
  });
  if (!res.response.ok) {
    const msg = await readResponseError(res, "Failed to list audit events");
    throw new Error(msg);
  }
  return res.data as AdminAuditListResponse;
}

export type AdminInviteCreateResponse = {
  invite_id: string;
  invite_link: string;
  expires_at: string;
};
export async function adminInviteCreate(body: {
  email: string;
  roles: string[];
  days_to_expire: number;
  email_user: boolean;
}): Promise<AdminInviteCreateResponse> {
  const http = (forge as unknown as { getFetch: () => typeof fetch }).getFetch();
  const res = await http(`/api/v1/admin/invites`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
    credentials: "include",
  });
  if (!res.ok) {
    const txt = await res.text().catch(() => "");
    throw new Error(txt || `${res.status}`);
  }
  return (await res.json()) as AdminInviteCreateResponse;
}

export type AdminAccessRequestListResponse =
  components["schemas"]["auth.AccessRequestListResponse"];
export async function adminListAccessRequests(params: {
  status?: string;
  q?: string;
  limit?: number;
  offset?: number;
}): Promise<AdminAccessRequestListResponse> {
  const res = await forge.raw.GET("/api/v1/admin/access-requests", {
    params: {
      query: { status: params.status, q: params.q, limit: params.limit, offset: params.offset },
    },
  });
  if (!res.response.ok) {
    const msg = await readResponseError(res, "Failed to load access requests");
    throw new Error(msg);
  }
  return res.data as AdminAccessRequestListResponse;
}

export async function adminDecideAccessRequest(
  id: string,
  body: components["schemas"]["auth.AccessRequestDecideRequest"]
): Promise<void> {
  const res = await forge.raw.PATCH("/api/v1/admin/access-requests/{id}", {
    params: { path: { id } },
    body: body as unknown as Record<string, never>,
  });
  if (res.response.status !== 204) {
    const msg = await readResponseError(res, "Failed to update access request");
    throw new Error(msg);
  }
}

export type RecoveryGenerateResponse = components["schemas"]["auth.RecoveryGenerateResponse"];
export async function adminGenerateUserRecoveryCodes(
  userId: string
): Promise<RecoveryGenerateResponse> {
  const res = await forge.raw.POST("/api/v1/admin/users/{id}/recovery/codes/generate", {
    params: { path: { id: userId } },
  });
  if (!res.response.ok) {
    const msg = await readResponseError(res, "Failed to generate recovery codes");
    throw new Error(msg);
  }
  return res.data as RecoveryGenerateResponse;
}
