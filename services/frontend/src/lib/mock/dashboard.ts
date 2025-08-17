// Deterministic dashboard mocks and simple role-based permission helper
import { addMinutes, subHours, subMinutes, formatISO } from "date-fns";

export type Kpi = {
  id: string;
  label: string;
  value: number;
  unit?: string;
  spark: number[]; // last 7d or similar small window
  badge: "good" | "warn" | "bad";
  hint?: string; // e.g., "vs 7d avg"
};

export type EnvHealth = {
  id: "dev" | "preprod" | "prod";
  release: string;
  lastDeployAt: string; // ISO
  drift: boolean;
  errorRate: number; // percentage 0-100
};

export type ActivityItem = {
  id: string;
  type: "deploy" | "job";
  name: string;
  env: "dev" | "preprod" | "prod";
  status: "queued" | "running" | "failed" | "succeeded";
  startedAt: string; // ISO
  finishedAt?: string; // ISO
  durationMs?: number;
};

// Internal stable seed data
const baseNow = new Date();

const baseSpark = [82, 85, 80, 88, 90, 92, 91, 93, 94, 95, 94];

function computeBadge(v: number, warnBelow: number, badBelow: number): "good" | "warn" | "bad" {
  if (v < badBelow) return "bad";
  if (v < warnBelow) return "warn";
  return "good";
}

export function getKpis(): Kpi[] {
  // Derive slight, deterministic variation from current minute
  const minute = new Date().getMinutes();
  const mod = (minute % 5) - 2; // -2..+2
  const success24h = 96 + mod; // 94..98
  const previewEnvs = 7 + ((minute % 3) - 1); // 6..8
  const queueLen = Math.max(0, 3 + ((minute % 4) - 2)); // 1..5
  const secretsOverdue = 4; // stable
  const certsExpiring = 2 + (minute % 2); // 2..3

  return [
    {
      id: "deploy_success",
      label: "Deploy success (24h)",
      value: success24h,
      unit: "%",
      spark: baseSpark,
      badge: computeBadge(success24h, 95, 92),
      hint: "vs 7d avg",
    },
    {
      id: "preview_envs",
      label: "Active preview envs",
      value: previewEnvs,
      spark: [6, 6, 7, 7, 8, 8, 7],
      badge: computeBadge(100 - previewEnvs, 94, 90),
      hint: "oldest 3d",
    },
    {
      id: "build_queue",
      label: "Build queue length",
      value: queueLen,
      spark: [2, 3, 2, 4, 5, 3, 2],
      badge: computeBadge(100 - queueLen * 10, 85, 70),
      hint: "ETA ~8m",
    },
    {
      id: "secrets_overdue",
      label: "Secrets overdue",
      value: secretsOverdue,
      spark: [5, 4, 4, 3, 4, 4, 4],
      badge: secretsOverdue > 3 ? "warn" : "good",
      hint: "rotate now",
    },
    {
      id: "certs_expiring",
      label: "Certs expiring ≤14d",
      value: certsExpiring,
      spark: [1, 1, 2, 2, 2, 3, 3],
      badge: certsExpiring >= 3 ? "warn" : "good",
      hint: "renew",
    },
  ];
}

export function getEnvironments(): EnvHealth[] {
  const now = new Date();
  return [
    {
      id: "dev",
      release: "v1.8.4",
      lastDeployAt: formatISO(subMinutes(now, 28)),
      drift: false,
      errorRate: 0.3,
    },
    {
      id: "preprod",
      release: "v1.8.3",
      lastDeployAt: formatISO(subHours(now, 3)),
      drift: Math.abs(now.getMinutes() % 7) === 0, // occasional drift
      errorRate: 0.7,
    },
    {
      id: "prod",
      release: "v1.8.2",
      lastDeployAt: formatISO(subHours(now, 26)),
      drift: false,
      errorRate: 0.2,
    },
  ];
}

const baseActivities: ActivityItem[] = Array.from({ length: 20 }).map((_, i) => {
  const start = subMinutes(baseNow, i * 7 + 3);
  const env: EnvHealth["id"] = i % 3 === 0 ? "dev" : i % 3 === 1 ? "preprod" : "prod";
  const type: ActivityItem["type"] = i % 2 === 0 ? "deploy" : "job";
  const name = type === "deploy" ? `Deploy ${env} #${142 + i}` : `Job build-${i}`;
  return {
    id: `act_${i}`,
    type,
    name,
    env,
    status: "queued",
    startedAt: formatISO(start),
  } as ActivityItem;
});

export function getActivity(): ActivityItem[] {
  const now = new Date();
  return baseActivities.map((a, idx) => {
    const minutesAgo = Math.max(0, (now.getTime() - new Date(a.startedAt).getTime()) / 60000);
    let status: ActivityItem["status"] = "queued";
    let finishedAt: string | undefined = undefined;
    if (minutesAgo > 2 && minutesAgo <= 6) status = "running";
    else if (minutesAgo > 6) {
      status = idx % 7 === 0 ? "failed" : "succeeded";
      finishedAt = formatISO(addMinutes(new Date(a.startedAt), 6));
    }
    const durationMs = finishedAt
      ? new Date(finishedAt).getTime() - new Date(a.startedAt).getTime()
      : undefined;
    return { ...a, status, finishedAt, durationMs };
  });
}

// Simple role + step-up model for gating actions
export type PermissionResult = { allowed: boolean; reason?: string };

type ActionKey =
  | "promote:dev"
  | "promote:preprod"
  | "promote:prod"
  | "create:preview"
  | "release:create"
  | "cert:request"
  | "secret:rotate";

const currentUser = { role: "developer" as "developer" | "viewer" | "admin", stepUp: false };

export function can(action: ActionKey): PermissionResult {
  // viewers can do nothing
  if (currentUser.role === "viewer") return { allowed: false, reason: "Viewer role" };
  if (action === "promote:prod") return { allowed: false, reason: "Step-up required" };
  // developers allowed for rest; admins allowed for all
  return { allowed: true };
}
