import { useMemo, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { LogoMark } from "@/components/brand/LogoMark";
import { useToast } from "@/hooks/use-toast";
import { usePageTitle } from "@/hooks/usePageTitle";
import { parseInviteToken, formatExpiry } from "@/lib/invite";
import { withLatency } from "@/mocks/latency";
import { useAppStore } from "@/store/app-store";

export default function InviteLanding() {
  const { token } = useParams<{ token: string }>();
  // const navigate = useNavigate();
  const { toast } = useToast();
  const [busy, setBusy] = useState(false);
  const { actions } = useAppStore();

  const invite = useMemo(() => parseInviteToken(token), [token]);

  const isValid = invite.status ==="valid";
  const seo = usePageTitle(
    isValid ? "Accept invite • Catalyst Forge" : "Invite unavailable • Catalyst Forge",
    isValid
      ? "Accept your Catalyst Forge invite and register your device for passwordless login."
      : "This invite is expired, used, or invalid. Request a new invite to continue.",
    "/invite"
  );

  async function handleAccept() {
    if (invite.status !== "valid") {
      toast({
        title: "Invite not valid",
        description: "Please request a new invite to continue.",
        variant: "destructive",
      });
      return;
    }
    setBusy(true);
    try {
      // Simulate WebAuthn registration ceremony (mock)
      await withLatency(400, 900);
      await actions.registerInitialDevice("Passkey (this device)");
    } catch {
      toast({
        title: "Registration failed",
        description: "Please try again or contact your admin.",
        variant: "destructive",
      });
    } finally {
      setBusy(false);
    }
  }

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
            {isValid ? "You’ve been invited to join Catalyst Forge" : "Your invite link has expired or is invalid"}
          </h1>
          {invite.status === "expired" && (
            <p className="mt-2 text-sm text-muted-foreground" aria-live="polite">This invite has expired.</p>
          )}
          {invite.status === "used" && (
            <p className="mt-2 text-sm text-muted-foreground" aria-live="polite">This invite was already used.</p>
          )}

          <div className="mt-8">
            {invite.status === "valid" ? (
              <div>
                <Button size="cta" className="w-full sm:w-auto" onClick={handleAccept} disabled={busy} variant="hero">
                  {busy ? "Registering device…" : "Register device"}
                </Button>
                <p className="mt-3 text-[13px] text-muted-foreground/75 dark:text-muted-foreground/65">
                  Registers this device for passwordless login.
                </p>

                <div className="mt-6 space-y-1 text-sm text-muted-foreground">
                  {(invite.invitedBy || invite.email) && (
                    <p>
                      {invite.invitedBy && (
                        <>
                          <span className="font-medium text-foreground/90">From:</span> {invite.invitedBy}
                        </>
                      )}
                      {invite.invitedBy && invite.email && <span className="mx-[0.5em]">•</span>}
                      {invite.email && (
                        <>
                          <span className="font-medium text-foreground/90">Sent to:</span>{" "}
                          <span>{invite.email}</span>
                        </>
                      )}
                    </p>
                  )}
                  {!invite.email && <p>We couldn’t determine the invite email.</p>}
                  {invite.role && (
                    <p>
                      <span className="font-medium text-foreground">Role:</span> {invite.role}
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <div>
                <div className="mt-4 flex flex-col items-center gap-3">
                  {invite.email ? (
                    <Button size="cta" asChild>
                      <Link to="/welcome?request=1" state={{ email: invite.email, auto: true }}>
                        Request a new invite
                      </Link>
                    </Button>
                  ) : (
                    <Button size="cta" asChild>
                      <Link to="/welcome?request=1">Request a new invite</Link>
                    </Button>
                  )}

                </div>
              </div>
            )}
          </div>

          {isValid ? (
            <p className="mt-8 text-xs text-muted-foreground">
              {invite.expiresAt ? (
                <>
                  Expires on {formatExpiry(invite.expiresAt)}. Need help?{" "}
                  <a className="underline-offset-4 hover:underline focus-visible:underline" href="mailto:admin@company.com">Contact your admin</a>.
                </>
              ) : (
                <>
                  Expires soon. Need help?{" "}
                  <a className="underline-offset-4 hover:underline focus-visible:underline" href="mailto:admin@company.com">Contact your admin</a>.
                </>
              )}
            </p>
          ) : (
            <p className="mt-8 text-xs text-muted-foreground">
              Need help? <a className="underline-offset-4 hover:underline focus-visible:underline" href="mailto:admin@company.com">Contact your admin</a>.
            </p>
          )}
        </section>
      </main>
    </div>
  );
}
