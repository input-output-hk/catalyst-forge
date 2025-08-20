import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { loginWithProvider } from "@/lib/auth/oidc";

type KratosError = {
    error?: {
        id?: string;
        code?: number;
        status?: string;
        reason?: string;
        message?: string;
        details?: unknown;
    };
};

export default function KratosError() {
    const [loading, setLoading] = useState(true);
    const [data, setData] = useState<KratosError | null>(null);
    const [loadError, setLoadError] = useState<string | null>(null);
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();

    const seo = usePageTitle(
        "Sign-in error – Catalyst Forge",
        "An error occurred while processing your sign-in or registration.",
        "/auth/error"
    );

    const provider = (import.meta.env.VITE_OIDC_PROVIDER as string) || "github";
    const providerLabel = useMemo(() => {
        const p = provider.toLowerCase();
        if (p === "google") return "Google";
        if (p === "github") return "GitHub";
        return provider;
    }, [provider]);
    const allowedDomains = useMemo(() => {
        const raw = (import.meta.env.VITE_ALLOWED_OIDC_DOMAINS as string) || "";
        return raw
            .split(",")
            .map((s) => s.trim())
            .filter((s) => s.length > 0);
    }, []);

    useEffect(() => {
        const id = searchParams.get("id");
        if (!id) {
            setLoadError("Missing error identifier.");
            setLoading(false);
            return;
        }
        async function run() {
            try {
                const resp = await fetch(
                    `/.ory/kratos/public/self-service/errors?id=${encodeURIComponent(id)}`,
                    { credentials: "include" }
                );
                if (!resp.ok) {
                    setLoadError(`Failed to load error (HTTP ${resp.status}).`);
                    return;
                }
                const json = (await resp.json()) as KratosError;
                setData(json);
            } catch (err) {
                setLoadError("Failed to load error details.");
            } finally {
                setLoading(false);
            }
        }
        run();
    }, [searchParams]);

    return (
        <div className="container mx-auto max-w-2xl py-16 px-4">
            {seo}
            <div className="space-y-6">
                <div>
                    <h1 className="text-2xl font-bold tracking-tight">We couldn't complete that request</h1>
                    <p className="text-muted-foreground mt-1">
                        Something went wrong while processing your sign-in or registration. You can retry
                        below, or return to the welcome page.
                    </p>
                </div>

                {(() => {
                    const msg = (data?.error?.message || "").toLowerCase();
                    const domainRestricted =
                        msg.includes("jsonnetsecure") &&
                        (msg.includes("restricted to specific organization domains") ||
                            msg.includes("restricted to specific organisation domains"));
                    if (!domainRestricted) return null;
                    return (
                        <div className="rounded-md border border-warning/30 bg-warning/10 p-4">
                            <p className="text-sm">This account's email domain isn't allowed for sign-in.</p>
                            {allowedDomains.length > 0 ? (
                                <p className="text-sm mt-1">
                                    Please use your organization account. Allowed domains: {allowedDomains.join(", ")}
                                </p>
                            ) : (
                                <p className="text-sm mt-1">Please use your organization {providerLabel} account.</p>
                            )}
                        </div>
                    );
                })()}

                <div className="flex flex-wrap gap-2">
                    <Button onClick={() => loginWithProvider(provider, "/")}>
                        Try again with {providerLabel}
                    </Button>
                    <Button variant="outline" onClick={() => navigate("/welcome", { replace: true })}>
                        Back to Welcome
                    </Button>
                    <Button asChild variant="ghost">
                        <Link to="/">Go to Dashboard</Link>
                    </Button>
                </div>

                <div className="rounded-md border bg-muted/40 p-4">
                    {loading ? (
                        <p className="text-sm text-muted-foreground">Loading error details…</p>
                    ) : loadError ? (
                        <p className="text-sm text-destructive">{loadError}</p>
                    ) : (
                        <div className="space-y-2">
                            <p className="text-sm">
                                <span className="font-medium">Message: </span>
                                {data?.error?.message || data?.error?.reason || "Unknown error"}
                            </p>
                            {data?.error?.status && (
                                <p className="text-sm">
                                    <span className="font-medium">Status: </span>
                                    {data.error.status}
                                </p>
                            )}
                            {data?.error?.code && (
                                <p className="text-sm">
                                    <span className="font-medium">Code: </span>
                                    {data.error.code}
                                </p>
                            )}
                            <details className="mt-2">
                                <summary className="cursor-pointer text-sm">Technical details</summary>
                                <pre className="mt-2 whitespace-pre-wrap text-xs">
                                    {JSON.stringify(data, null, 2)}
                                </pre>
                            </details>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}


