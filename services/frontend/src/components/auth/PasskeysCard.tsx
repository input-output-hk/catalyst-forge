import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Configuration, FrontendApi } from "@ory/client";
import { getKratosPublicBaseUrl } from "@/lib/auth/kratos";

type UiNode = {
    group?: string;
    attributes?: {
        name?: string;
        type?: string;
        value?: unknown;
        disabled?: boolean;
    };
    meta?: { label?: { text?: string } };
};

type SettingsFlowLike = {
    id?: string;
    ui?: {
        action?: string;
        method?: string;
        nodes?: UiNode[];
    };
};

const kratosBase = getKratosPublicBaseUrl();
const kratos = new FrontendApi(
    new Configuration({ basePath: kratosBase, baseOptions: { withCredentials: true } })
);

function ensureWebAuthnScript(onReady?: () => void): void {
    const id = "ory-webauthn-js";
    if (document.getElementById(id)) return;
    const script = document.createElement("script");
    script.id = id;
    script.async = true;
    script.src = `${kratosBase}/.well-known/ory/webauthn.js`;
    if (onReady) script.onload = onReady;
    document.head.appendChild(script);
}

export default function PasskeysCard() {
    const [flow, setFlow] = useState<SettingsFlowLike | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        ensureWebAuthnScript(() => {
            (window as unknown as { oryWebAuthn?: { bind?: () => void } }).oryWebAuthn?.bind?.();
        });
        let cancelled = false;
        async function start() {
            try {
                setLoading(true);
                const { data } = await kratos.createBrowserSettingsFlow();
                if (!cancelled) setFlow(data as unknown as SettingsFlowLike);
            } catch (e) {
                if (!cancelled) setError("Failed to load passkeys settings");
            } finally {
                if (!cancelled) setLoading(false);
            }
        }
        start();
        return () => {
            cancelled = true;
        };
    }, []);

    const passkeyGroup = useMemo(() => {
        const groups = new Set((flow?.ui?.nodes ?? []).map((n) => (n.group ?? "").toLowerCase()));
        return groups.has("passkey") ? "passkey" : "webauthn"; // support older configs
    }, [flow]);

    const passkeyNodes = useMemo(() => {
        const nodes = flow?.ui?.nodes ?? [];
        return nodes.filter((n) => (n.group ?? "").toLowerCase() === passkeyGroup);
    }, [flow, passkeyGroup]);

    const csrfValue = useMemo(() => {
        const nodes = flow?.ui?.nodes ?? [];
        const match = nodes.find((n) => n.attributes?.name === "csrf_token");
        const raw = match?.attributes?.value;
        return typeof raw === "string" ? raw : raw != null ? String(raw) : undefined;
    }, [flow]);

    const registerNode = useMemo(() => {
        return passkeyNodes.find((n) => {
            const name = (n.attributes?.name ?? "").toLowerCase();
            const type = (n.attributes?.type ?? "").toLowerCase();
            return type === "button" && (name === "passkey_register" || name === "webauthn_register");
        });
    }, [passkeyNodes]);

    // After nodes are rendered, ask the binder to attach handlers
    useEffect(() => {
        (window as unknown as { oryWebAuthn?: { bind?: () => void } }).oryWebAuthn?.bind?.();
    }, [registerNode]);

    // Debug helper: expose flow and group for console inspection
    useEffect(() => {
        if (flow) {
            (window as unknown as { cfPasskeysFlow?: unknown }).cfPasskeysFlow = flow;
            // eslint-disable-next-line no-console
            console.log("[Passkeys] group:", passkeyGroup, "nodes:", flow.ui?.nodes);
        }
    }, [flow, passkeyGroup]);

    if (loading) return <div className="text-sm text-muted-foreground">Loading…</div>;
    if (error) return <div className="text-sm text-destructive">{error}</div>;
    if (!flow) return null;

    const action = flow.ui?.action ?? `${kratosBase}/self-service/settings?flow=` + (flow.id ?? "");
    const method = (flow.ui?.method ?? "POST").toUpperCase();

    return (
        <form action={action} method={method} className="space-y-3">
            {csrfValue && <input type="hidden" name="csrf_token" value={csrfValue} />}
            {/* Ensure method is present when rendering a subset */}
            {!passkeyNodes.some((n) => (n.attributes?.name ?? "").toLowerCase() === "method") && (
                <input
                    type="hidden"
                    name="method"
                    value={passkeyGroup === "passkey" ? "passkey" : "webauthn"}
                />
            )}
            {/* Render passkey/webauthn group nodes */}
            {passkeyNodes.map((n, idx) => {
                const type = (n.attributes?.type ?? "").toLowerCase();
                const name = n.attributes?.name ?? "";
                const value = (n.attributes?.value as string | undefined) ?? "";
                const disabled = Boolean(n.attributes?.disabled);

                if (type === "button") {
                    const triggerName = (n.attributes as unknown as { onclickTrigger?: string; onclick?: string })?.onclickTrigger;
                    return (
                        <button
                            key={`btn-${name}-${idx}`}
                            type="button"
                            name={name}
                            value={value}
                            disabled={disabled}
                            className="inline-flex h-9 items-center rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground hover:opacity-95 focus:outline-none focus:ring-2 focus:ring-ring"
                            onClick={() => {
                                // If script loaded late, attempt a bind before click handling
                                (window as unknown as { oryWebAuthn?: { bind?: () => void } }).oryWebAuthn?.bind?.();
                                // Call the trigger function exposed by ory script
                                const w = window as unknown as Record<string, unknown>;
                                const fn = triggerName && typeof w[triggerName] === "function" ? (w[triggerName] as () => void) : undefined;
                                fn?.();
                            }}
                        >
                            {n.meta?.label?.text || "Register passkey"}
                        </button>
                    );
                }

                // Render remove controls as visible submit buttons
                if (type === "submit" && (name === "passkey_remove" || name === "webauthn_remove")) {
                    const label = n.meta?.label?.text || "Remove";
                    return (
                        <button
                            key={`rm-${name}-${value}-${idx}`}
                            type="submit"
                            name={name}
                            value={value}
                            disabled={disabled}
                            className="inline-flex h-8 items-center rounded-md border px-2 text-xs font-medium hover:bg-accent"
                        >
                            {label}
                        </button>
                    );
                }

                // Hidden inputs such as method fields/data
                return <input key={`inp-${name}-${idx}`} type={type || "hidden"} name={name} value={value} hidden />;
            })}
            {passkeyNodes.length === 0 && (
                <div className="text-sm text-muted-foreground">
                    No passkey nodes returned from Kratos. Check that passkey is enabled and the settings flow includes the passkey group.
                </div>
            )}
        </form>
    );
}


