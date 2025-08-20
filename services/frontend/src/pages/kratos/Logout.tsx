import { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAppStore } from "@/store/app-store";

export default function KratosLogout() {
    const navigate = useNavigate();
    const location = useLocation();
    const { actions } = useAppStore();

    useEffect(() => {
        async function run() {
            try {
                // Ask Kratos for logout URL (browser flow), then follow it to clear cookies
                const resp = await fetch("/.ory/kratos/public/self-service/logout/browser", {
                    credentials: "include",
                });
                if (resp.ok) {
                    const data = await resp.json();
                    const redirectTo: string | undefined = data.logout_url;
                    actions.logout();
                    if (redirectTo) {
                        window.location.replace(redirectTo);
                        return;
                    }
                }
            } finally {
                const from = (location.state as { from?: string })?.from;
                navigate("/welcome", { replace: true, state: { from } });
            }
        }
        run();
    }, [navigate, location, actions]);
    return null;
}


