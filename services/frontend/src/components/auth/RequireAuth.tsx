import { useEffect, useState } from "react";
import { Outlet, useLocation, Navigate } from "react-router-dom";
import { useAppStore } from "@/store/app-store";
import { sendEmailVerificationIfNeeded } from "@/lib/auth/verification";

export default function RequireAuth() {
  const { state, actions } = useAppStore();
  const [checking, setChecking] = useState(true);
  const location = useLocation();

  useEffect(() => {
    let cancelled = false;
    async function ensure() {
      try {
        if (!state.session.authed) {
          async function tryWhoAmI(): Promise<boolean> {
            const whoami = await fetch("/.ory/kratos/public/sessions/whoami", {
              credentials: "include",
            });
            if (whoami.ok) {
              const data = await whoami.json();
              const email = data?.identity?.traits?.email ?? "user";
              // Roles now come from authorization (Keto / Oathkeeper), not Kratos session.
              actions.login(email, Array.isArray(state.session.roles) ? state.session.roles : []);
              // Fire-and-forget: if email is unverified, trigger verification email
              void sendEmailVerificationIfNeeded(data?.identity);
              return true;
            }
            return false;
          }

          let ok = await tryWhoAmI();
          if (!ok && !cancelled) {
            await new Promise((r) => setTimeout(r, 500));
            if (!cancelled) {
              ok = await tryWhoAmI();
            }
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
  }, [state.session.authed, state.session.roles, actions]);

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
