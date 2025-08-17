import { useEffect, useState } from "react";
import { Outlet, useLocation, Navigate } from "react-router-dom";
import { authApi } from "@/features/auth/api/queries";
// Import OpenAPI types from vendored client types
import type { paths } from "forge-client";
import { useAppStore } from "@/store/app-store";

export default function RequireAuth() {
  const { state, actions } = useAppStore();
  const [checking, setChecking] = useState(true);
  const location = useLocation();

  useEffect(() => {
    let cancelled = false;
    async function ensure() {
      try {
        if (!state.session.authed) {
          const res = await authApi.me();
          if (res.response.ok && res.data) {
            type MeResponse =
              paths["/api/v1/auth/me"]["get"]["responses"][200]["content"]["application/json"];
            const me: MeResponse = res.data as MeResponse;
            const email = me.email || "user";
            const roles: string[] = Array.isArray(me.roles) ? (me.roles as string[]) : [];
            actions.login(email, roles);
          }
        }
      } finally {
        if (!cancelled) setChecking(false);
      }
    }
    ensure();
    return () => {
      cancelled = true;
    };
  }, [state.session.authed, actions]);

  if (checking) return null;

  if (!state.session.authed) {
    return <Navigate to="/welcome" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}

export function RequireAdmin({ children }: { children: React.ReactNode }) {
  const { state } = useAppStore();
  const location = useLocation();
  const isAdmin = Array.isArray(state.session.roles) && state.session.roles.includes("admin");
  if (!isAdmin) {
    return <Navigate to="/" replace state={{ from: location.pathname }} />;
  }
  return <>{children}</>;
}
