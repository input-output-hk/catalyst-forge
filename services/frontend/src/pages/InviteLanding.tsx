import { useEffect, useMemo, useState } from "react";
import { useParams, Link, useNavigate, useLocation } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { LogoMark } from "@/components/brand/LogoMark";
import { useToast } from "@/hooks/use-toast";
import { usePageTitle } from "@/hooks/usePageTitle";
import { parseInviteToken, formatExpiry } from "@/lib/auth/invite";
import { useAppStore } from "@/store/app-store";
import { refreshAccessToken, extractErrorMessage, readResponseError } from "@/lib/api";
import { forge } from "@/lib/client";
import {
  extractCreationOptionsFromServer,
  decodeCreationOptions,
  encodeAttestation,
  createCredentialFromServerPublicKey,
} from "@/lib/webauthn";
import type { components } from "forge-client";
import { requestCodesAndStartGate } from "@/lib/auth/recovery";

// Type definitions for cleaner code
type InvitePreviewResponse = components["schemas"]["auth.InvitePreviewResponse"];
type OnboardBeginRequest = components["schemas"]["auth.OnboardBeginRequest"];
type OnboardBeginResponse = components["schemas"]["auth.OnboardBeginResponse"];
type OnboardCompleteRequest = components["schemas"]["auth.OnboardCompleteRequest"];
type RecoveryGenerateResponse = components["schemas"]["auth.RecoveryGenerateResponse"];

interface ErrorWithMessage {
  message?: string;
}

export default function InviteLanding() {
  const { token } = useParams<{ token: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [busy, setBusy] = useState(false);
  const { actions } = useAppStore();

  // Meta info (best-effort) from token for display only
  const inviteMeta = useMemo(() => parseInviteToken(token), [token]);
  const isValid = Boolean(token); // Treat presence of token as potentially valid; backend will verify

  const seo = usePageTitle(
    isValid ? "Accept invite • Catalyst Forge" : "Invite unavailable • Catalyst Forge",
    isValid
      ? "Accept your Catalyst Forge invite and register your device for passwordless login."
      : "This invite is expired, used, or invalid. Request a new invite to continue.",
    "/invite"
  );

  // Fetch preview to get email/expiry for display
  const [previewEmail, setPreviewEmail] = useState<string | null>(null);
  const [previewExpiry, setPreviewExpiry] = useState<string | null>(null);

  useEffect(() => {
    async function fetchInvitePreview() {
      if (!token) return;
      try {
        const id = new URLSearchParams(location.search).get("id") || undefined;
        const res = await forge.raw.GET("/api/v1/admin/invites/preview", {
          params: { query: { token, id } },
        });
        if (res.response.ok) {
          const j = res.data as InvitePreviewResponse;
          if (j?.email) setPreviewEmail(j.email);
          if (j?.expires_at) setPreviewExpiry(j.expires_at);
        }
      } catch {
        // ignore preview fetch errors
      }
    }
    fetchInvitePreview();
  }, [token, location.search]);

  // Begin onboarding on accept
  async function handleAccept() {
    if (!token) {
      toast({
        title: "Invite not found",
        description: "This link is missing a token.",
        variant: "destructive",
      });
      return;
    }

    setBusy(true);

    try {
      // Step 1: Begin onboarding process
      const inviteId = new URLSearchParams(location.search).get("id") || undefined;
      const beginRequestBody: OnboardBeginRequest = {
        invite_id: inviteId,
        token,
        device_name: "Passkey (this device)",
      };

      const beginResponse = await forge.raw.POST("/api/v1/auth/onboard/begin", {
        body: beginRequestBody,
      });

      if (!beginResponse.response.ok) {
        const errorText = await readResponseError(beginResponse, "");
        throw new Error(errorText || `Begin failed (${beginResponse.response.status})`);
      }

      const beginData = beginResponse.data as OnboardBeginResponse;

      // Step 2-3: Normalize server options and create WebAuthn credential
      const publicKeyCredential = await createCredentialFromServerPublicKey(
        beginData?.publicKey as unknown
      );

      if (!publicKeyCredential) {
        throw new Error("Credential creation cancelled");
      }

      // Step 4: Encode the credential for server
      const encodedCredential = encodeAttestation(publicKeyCredential);

      // Step 5: Determine the invite_id for completion
      const responseData = beginResponse.data as Record<string, unknown>;
      const responseInviteId = responseData?.invite_id as string | undefined;
      const finalInviteId = responseInviteId || inviteId || "";

      // Step 6: Complete the onboarding
      const completeRequestBody: OnboardCompleteRequest = {
        session_key: beginData.session_key,
        credential: encodedCredential,
        invite_id: finalInviteId,
      };

      const completeResponse = await forge.raw.POST("/api/v1/auth/onboard/complete", {
        body: completeRequestBody,
      });

      if (!completeResponse.response.ok) {
        const errorText = await readResponseError(completeResponse, "");
        throw new Error(errorText || `Complete failed (${completeResponse.response.status})`);
      }

      // Step 7: Refresh access token now that cookies are set
      await refreshAccessToken();

      // Step 8: Generate recovery keys and open gate
      await requestCodesAndStartGate(actions.startRecoveryGate, "/");

      // Step 9: Navigate to welcome page
      navigate("/welcome", { replace: true });
    } catch (error: unknown) {
      const errorMessage = extractErrorMessage(error, "Please try again or contact your admin.");
      toast({
        title: "Registration failed",
        description: errorMessage,
        variant: "destructive",
      });
    } finally {
      setBusy(false);
    }
  }

  // (shared error helpers are in lib/api.ts)

  return (
    <div className="relative min-h-dvh">
      {seo}
      <header className="absolute inset-x-0 top-0 z-10 flex items-center justify-between px-6 py-5">
        <div className="flex items-center gap-3">
          <LogoMark size={28} rounded />
          <span className="text-sm font-medium text-muted-foreground">Catalyst Forge</span>
        </div>
      </header>

      <main className="mx-auto flex min-h-dvh w-full max-w-2xl flex-col items-center justify-center px-6 text-center">
        <section className="w-full rounded-xl border bg-card/60 p-8 backdrop-blur">
          <h1 className="text-2xl font-semibold tracking-tight">
            {isValid
              ? "You’ve been invited to join Catalyst Forge"
              : "Invite unavailable • Catalyst Forge"}
          </h1>

          <div className="mt-8">
            {isValid ? (
              <div>
                <Button
                  size="cta"
                  className="w-full sm:w-auto"
                  onClick={handleAccept}
                  disabled={busy}
                  variant="hero"
                >
                  {busy ? "Registering device…" : "Register device"}
                </Button>
                <p className="mt-3 text-[13px] text-muted-foreground/75 dark:text-muted-foreground/65">
                  Registers this device for passwordless login.
                </p>

                <div className="mt-6 space-y-1 text-sm text-muted-foreground">
                  {(inviteMeta.invitedBy || inviteMeta.email || previewEmail) && (
                    <p>
                      {inviteMeta.invitedBy && (
                        <>
                          <span className="font-medium text-foreground/90">From:</span>{" "}
                          {inviteMeta.invitedBy}
                        </>
                      )}
                      {inviteMeta.invitedBy && (inviteMeta.email || previewEmail) && (
                        <span className="mx-[0.5em]">•</span>
                      )}
                      {(inviteMeta.email || previewEmail) && (
                        <>
                          <span className="font-medium text-foreground/90">Sent to:</span>{" "}
                          <span>{inviteMeta.email || previewEmail}</span>
                        </>
                      )}
                    </p>
                  )}
                  {!inviteMeta.email && !previewEmail && (
                    <p>We couldn’t determine the invite email.</p>
                  )}
                  {(inviteMeta.expiresAt || previewExpiry) && (
                    <p>
                      <span className="font-medium text-foreground/90">Expires on:</span>{" "}
                      <span>
                        {formatExpiry(
                          inviteMeta.expiresAt ||
                            (previewExpiry ? new Date(previewExpiry) : undefined)
                        )}
                      </span>
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <div>
                <div className="mt-4 flex flex-col items-center gap-3">
                  <Button size="cta" asChild>
                    <Link to="/welcome?request=1">Request a new invite</Link>
                  </Button>
                </div>
              </div>
            )}
          </div>

          <p className="mt-8 text-xs text-muted-foreground">
            Need help?{" "}
            <a
              className="underline-offset-4 hover:underline focus-visible:underline"
              href="mailto:admin@company.com"
            >
              Contact your admin
            </a>
            .
          </p>
        </section>
      </main>
    </div>
  );
}
