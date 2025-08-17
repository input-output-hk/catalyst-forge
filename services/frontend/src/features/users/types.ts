export type Credential = {
  id: string;
  label: string;
  addedAt: string;
  lastUsedAt?: string;
  platform: "platform" | "security_key";
  aaguid?: string;
};

export type User = {
  id: string;
  name: string;
  email: string;
  role: "admin" | "member";
  status: "active" | "disabled" | "pending_invite" | "pending_approval";
  createdAt: string;
  lastActivityAt?: string;
  sessions: number;
  credentials: Credential[];
  invites?: {
    link: string;
    expiresAt: string;
    lastSentAt: string;
  } | null;
  requests?: Array<{
    submittedAt: string;
    reason?: string;
    status: "pending" | "approved" | "rejected";
    decidedAt?: string;
    decidedBy?: string;
    note?: string;
  }>;
};
