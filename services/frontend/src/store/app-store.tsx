import { createContext, useCallback, useContext, useMemo, useState } from "react";
import {
  fixtures,
  type Service,
  type Environment,
  type JobRun,
  type SecretItem,
  type AuditEvent,
  type Device,
} from "@/mocks/fixtures";
import { withLatency } from "@/mocks/latency";

export type FeatureFlags = {
  darkMode: boolean;
  compact: boolean;
  streamLogs: boolean;
};

export type AppState = {
  services: Service[];
  environments: Environment[];
  jobs: JobRun[];
  secrets: SecretItem[];
  audit: AuditEvent[];
  devices: Device[];
  flags: FeatureFlags;
  session: { authed: boolean; user: string | null; roles: string[] };
};

const AppStoreCtx = createContext<{
  state: AppState;
  actions: {
    toggleFlag: (key: keyof FeatureFlags) => void;
    rotateSecret: (id: string) => Promise<void>;
    revealSecret: (id: string) => Promise<string>;
    addDevice: (label: string) => Promise<void>;
    registerInitialDevice: (label: string) => Promise<void>;
    removeDevice: (id: string) => Promise<void>;
    addAudit: (
      e: Omit<AuditEvent, "id" | "timestamp"> & { meta?: Record<string, unknown> }
    ) => void;
    login: (user: string, roles: string[]) => void;
    logout: () => void;
  };
} | null>(null);

export const AppStoreProvider = ({ children }: { children: React.ReactNode }) => {
  const [state, setState] = useState<AppState>(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem("cf-theme") : null;
    const prefersDark =
      typeof window !== "undefined" && window.matchMedia
        ? window.matchMedia("(prefers-color-scheme: dark)").matches
        : false;
    const dark = stored ? stored === "dark" : prefersDark;
    if (dark) {
      document.documentElement.classList.add("dark");
    }
    return {
      services: fixtures.services,
      environments: fixtures.environments,
      jobs: fixtures.jobs,
      secrets: fixtures.secrets,
      audit: fixtures.audit,
      devices: fixtures.devices,
      flags: { darkMode: dark, compact: false, streamLogs: true },
      session: { authed: false, user: null, roles: [] },

    };
  });

  const toggleFlag = useCallback((key: keyof FeatureFlags) => {
    setState((s) => {
      const nextVal = !s.flags[key];
      const next = { ...s, flags: { ...s.flags, [key]: nextVal } };
      if (key === "darkMode") {
        const el = document.documentElement;
        el.classList.toggle("dark", Boolean(nextVal));
        try {
          localStorage.setItem("cf-theme", nextVal ? "dark" : "light");
        } catch {
          // Ignore localStorage errors
        }
      }
      return next;
    });
  }, []);

  const addAudit = useCallback(
    (evt: Omit<AuditEvent, "id" | "timestamp"> & { meta?: Record<string, unknown> }) => {
      setState((s) => ({
        ...s,
        audit: [
          {
            id: `evt_${Date.now()}`,
            timestamp: new Date().toISOString(),
            ...evt,
          },
          ...s.audit,
        ],
      }));
    },
    []
  );

  const rotateSecret = useCallback(
    async (id: string) => {
      await withLatency();
      setState((s) => ({
        ...s,
        secrets: s.secrets.map((sec) =>
          sec.id === id
            ? { ...sec, version: sec.version + 1, lastRotatedAt: new Date().toISOString() }
            : sec
        ),
      }));
      addAudit({ actor: state.session.user ?? "user", action: "secret.rotate", resource: id });
    },
    [addAudit, state.session.user]
  );

  const revealSecret = useCallback(
    async (id: string) => {
      await withLatency();
      const sec = state.secrets.find((s) => s.id === id)!;
      addAudit({ actor: state.session.user ?? "user", action: "secret.reveal", resource: id });
      return sec.value;
    },
    [state.secrets, state.session.user, addAudit]
  );

  const addDevice = useCallback(
    async (label: string) => {
      await withLatency();
      setState((s) => ({
        ...s,
        devices: [
          ...s.devices,
          { id: `dev_${Date.now()}`, label, addedAt: new Date().toISOString() },
        ],
      }));
      addAudit({ actor: state.session.user ?? "user", action: "device.add", resource: label });
    },
    [addAudit, state.session.user]
  );

  const registerInitialDevice = useCallback(
    async (label: string) => {
      await withLatency();
      setState((s) => ({
        ...s,
        devices: [
          ...s.devices,
          { id: `dev_${Date.now()}`, label, addedAt: new Date().toISOString() },
        ],
      }));
      addAudit({ actor: state.session.user ?? "user", action: "device.register", resource: label });
    },
    [addAudit, state.session.user]
  );

  const removeDevice = useCallback(
    async (id: string) => {
      await withLatency();
      setState((s) => ({ ...s, devices: s.devices.filter((d) => d.id !== id) }));
      addAudit({ actor: state.session.user ?? "user", action: "device.remove", resource: id });
    },
    [addAudit, state.session.user]
  );

  const login = (user: string, roles: string[]) =>
    setState((s) => ({ ...s, session: { authed: true, user, roles } }));
  const logout = () =>
    setState((s) => ({ ...s, session: { authed: false, user: null, roles: [] } }));

  const value = useMemo(
    () => ({
      state,
      actions: {
        toggleFlag,
        rotateSecret,
        revealSecret,
        addDevice,
        registerInitialDevice,
        removeDevice,
        addAudit,
        login,
        logout,
      },
    }),
    [
      state,
      toggleFlag,
      rotateSecret,
      revealSecret,
      addDevice,
      registerInitialDevice,
      removeDevice,
      addAudit,
    ]
  );

  return <AppStoreCtx.Provider value={value}>{children}</AppStoreCtx.Provider>;
};

export function useAppStore() {
  const ctx = useContext(AppStoreCtx);
  if (!ctx) throw new Error("useAppStore must be used within AppStoreProvider");
  return ctx;
}
