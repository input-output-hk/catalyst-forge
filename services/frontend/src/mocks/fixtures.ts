export type Service = {
  id: string;
  name: string;
  owner: string;
  status: "healthy" | "degraded" | "down";
  envs: string[];
};
export type Environment = {
  id: string;
  name: string;
  region: string;
  status: "ready" | "provisioning" | "error";
};
export type JobRun = {
  id: string;
  serviceId: string;
  status: "running" | "success" | "failed";
  startedAt: string;
  durationSec?: number;
  logs?: string[];
};
export type SecretItem = {
  id: string;
  key: string;
  version: number;
  value: string;
  lastRotatedAt: string;
};
export type AuditEvent = {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  resource: string;
  meta?: Record<string, unknown>;
};
export type Device = { id: string; label: string; addedAt: string };

const now = Date.now();

export const fixtures = {
  services: [
    { id: "srv_web", name: "web", owner: "alice", status: "healthy", envs: ["prod", "staging"] },
    {
      id: "srv_api",
      name: "api",
      owner: "bob",
      status: "degraded",
      envs: ["prod", "staging", "dev"],
    },
    { id: "srv_worker", name: "worker", owner: "carol", status: "healthy", envs: ["prod"] },
  ] as Service[],
  environments: [
    { id: "env_prod", name: "prod", region: "us-east-1", status: "ready" },
    { id: "env_stg", name: "staging", region: "us-west-2", status: "ready" },
    { id: "env_dev", name: "dev", region: "eu-central-1", status: "provisioning" },
  ] as Environment[],
  jobs: [
    {
      id: "run_101",
      serviceId: "srv_api",
      status: "success",
      startedAt: new Date(now - 3600_000).toISOString(),
      durationSec: 125,
    },
    {
      id: "run_102",
      serviceId: "srv_api",
      status: "failed",
      startedAt: new Date(now - 7200_000).toISOString(),
      durationSec: 77,
    },
    {
      id: "run_103",
      serviceId: "srv_web",
      status: "running",
      startedAt: new Date(now - 60_000).toISOString(),
      logs: ["Booting...", "Connecting to DB..."],
    },
  ] as JobRun[],
  secrets: [
    {
      id: "sec_db",
      key: "DATABASE_URL",
      version: 3,
      value: "postgres://user:pass@db:5432/app",
      lastRotatedAt: new Date(now - 86_400_000).toISOString(),
    },
    {
      id: "sec_api",
      key: "API_TOKEN",
      version: 1,
      value: "sk_test_1234",
      lastRotatedAt: new Date(now - 604_800_000).toISOString(),
    },
  ] as SecretItem[],
  audit: [
    {
      id: "evt_1",
      timestamp: new Date(now - 10_000).toISOString(),
      actor: "system",
      action: "deploy",
      resource: "srv_web",
    },
  ] as AuditEvent[],
  devices: [
    {
      id: "dev_1",
      label: "MacBook Passkey",
      addedAt: new Date(now - 1000 * 60 * 60 * 24 * 12).toISOString(),
    },
  ] as Device[],
};
