import { useEffect } from "react";
import { forge } from "@/lib/client";
import { useAppStore } from "@/store/app-store";
import type { paths } from "forge-client";

export function AppBootstrap() {
    const { state, actions } = useAppStore();

    useEffect(() => {
        let cancelled = false;
        async function bootstrap() {
            try {
                if (!state.session.authed) {
                    // Attempt a silent refresh if cookies are present; ignore errors
                    try {
                        await forge.raw.POST("/api/v1/auth/refresh", { body: {} as Record<string, never> });
                    } catch { /* ignore */ }

                    // After refresh (or even if it failed), try to fetch profile
                    const me = await forge.raw.GET("/api/v1/auth/me");
                    if (!cancelled && me.response.ok && me.data) {
                        type MeResponse = paths["/api/v1/auth/me"]["get"]["responses"][200]["content"]["application/json"];
                        const m = me.data as MeResponse;
                        const email = m.email || "user";
                        const roles: string[] = Array.isArray(m.roles) ? m.roles : [];
                        actions.login(email, roles);
                    }
                }
            } catch { /* ignore */ }
        }
        bootstrap();
        return () => { cancelled = true; };
    }, [state.session.authed, actions]);

    return null;
}


