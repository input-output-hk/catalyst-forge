import { useEffect, useState } from "react";
import { Helmet } from "react-helmet-async";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { LogoMark } from "@/components/brand/LogoMark";
import { ForgeAnimation } from "@/components/brand/ForgeAnimation";
import { BRAND } from "@/lib/brand";
import { preloadAuthFlows } from "@/lib/preloaders";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { CheckCircle2, KeyRound, Info, AlertTriangle } from "lucide-react";
import RegisterRequestForm from "@/components/auth/RegisterRequestForm";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { useToast } from "@/hooks/use-toast";
import { useAppStore } from "@/store/app-store";
import { formatRecoveryKey, isValidRecoveryKey, normalizeRecoveryKeyInput } from "@/lib/recovery";
import { cn } from "@/lib/utils";
export default function Landing() {
  const navigate = useNavigate();
  const location = useLocation();
  const navState = (location.state as { email?: string; auto?: boolean } | null) || null;
  const [searchParams] = useSearchParams();
  const openRequested = searchParams.get("request") === "1";
  const openRecovery = searchParams.get("recover") === "1";
  const [requestOpen, setRequestOpen] = useState(false);
  const [recoveryOpen, setRecoveryOpen] = useState(false);
  const [submittedEmail, setSubmittedEmail] = useState<string | null>(null);
  const [jiggle, setJiggle] = useState(false);

  useEffect(() => {
    if (openRequested) setRequestOpen(true);
  }, [openRequested]);

  useEffect(() => {
    if (openRecovery) setRecoveryOpen(true);
  }, [openRecovery]);

  // Recovery form
  const recoverySchema = z.object({
    key: z
      .string()
      .min(1, "Recovery key is required")
      .refine((v) => isValidRecoveryKey(v), {
        message: "Use A–Z and 2–7; length 20, 25, or 26.",
      }),
  });
  type RecoveryForm = z.infer<typeof recoverySchema>;

  const recForm = useForm<RecoveryForm>({
    resolver: zodResolver(recoverySchema),
    defaultValues: { key: "" },
  });

  // Auto-focus recovery key when dialog opens
  useEffect(() => {
    if (recoveryOpen) {
      setTimeout(() => recForm.setFocus("key"), 10);
    }
  }, [recoveryOpen, recForm]);

  const { actions } = useAppStore();
  const { toast } = useToast();

  const onRecover = async (_values: RecoveryForm) => {
    actions.addAudit({ actor: "recovery", action: "recovery.verify", resource: "account", meta: { method: "key" } });
    toast({ title: "Recovery key accepted", description: "Registering this device..." });
    setRecoveryOpen(false);
    recForm.reset({ key: "" });
    navigate("/welcome", { replace: true });
    await actions.addDevice("This device");
  };

  // Pre-validation: length-based status with improved tone
  const keyValue = recForm.watch("key");
  const rawLen = normalizeRecoveryKeyInput(keyValue || "").length;
  const validLengths = [20, 25, 26] as const;
  const isAcceptedLen = validLengths.includes(rawLen as (typeof validLengths)[number]);
  const status = rawLen === 0
    ? null
    : rawLen < 20
      ? { tone: "warning" as const, msg: `Almost there — ${20 - rawLen} more characters needed (min 20).` }
      : isAcceptedLen
        ? { tone: "success" as const, msg: "Key format is valid." }
        : { tone: "warning" as const, msg: `Accepted lengths: 20, 25, or 26. Currently ${rawLen}.` };
  const fieldError = recForm.formState.errors.key?.message;

  return (
  <div className="relative min-h-screen bg-background text-foreground overflow-hidden">
      <Helmet>
        <title>Catalyst Forge – Developer Platform</title>
        <meta name="description" content="Developer platform landing: login, access request, and account recovery." />
        <link rel="canonical" href="/welcome" />
      </Helmet>

      {/* Animated brand gradient background */}
      <div className="absolute inset-0 bg-hero bg-hero-animated" aria-hidden="true" />

      {/* Optional forged pattern for subtle depth */}
      <div className="absolute inset-0 forged-pattern" aria-hidden="true" />

      {/* Cinematic ambient light sweep */}
      <div className="absolute inset-0 bg-light-sweep" aria-hidden="true" />

      {/* Vignette spotlight */}
      <div className="absolute inset-0 vignette-center" aria-hidden="true" />

      {/* Animated forge background */}
      <ForgeAnimation className="z-0" />

      <main className="relative z-10 flex min-h-screen items-center justify-center px-4">
        <section
          className="w-full max-w-xl panel-hero p-8 md:p-10"
          aria-label="Welcome to Catalyst Forge"
        >
          <header className="text-center space-y-6">
            <div className="logo-motif pulse-ambient mx-auto inline-flex items-center justify-center rounded-full p-3">
              <LogoMark size={78} rounded />
            </div>
            <div className="space-y-2">
              <h1 className="text-3xl md:text-4xl font-bold tracking-tight fade-in-up">
                Catalyst Forge
              </h1>
              <p className="text-muted-foreground text-base md:text-lg fade-in-up anim-delay-200">
                Forge your path, build your future.
              </p>
            </div>
          </header>

          <div className="mt-8 grid grid-cols-1 sm:grid-cols-2 gap-3 md:gap-4">
            <Button
              size="lg"
              variant="hero"
              className="btn-ripple hover-scale hover-glow w-full"
              aria-label="Login with this device"
              onPointerEnter={preloadAuthFlows}
              onFocus={preloadAuthFlows}
              onTouchStart={preloadAuthFlows}
              onClick={() => navigate("/auth-demo")}
            >
              Login with this device
            </Button>
            <Button
              size="lg"
              variant="outline"
              className="cta-outline-invert btn-ripple hover-scale hover-glow w-full"
              aria-label="Get access"
              onPointerEnter={preloadAuthFlows}
              onFocus={preloadAuthFlows}
              onTouchStart={preloadAuthFlows}
              onClick={() => setRequestOpen(true)}
            >
              Get access
            </Button>
          </div>

          <div className="mt-8 text-center text-xs text-muted-foreground/80">
            <button
              onClick={() => setRecoveryOpen(true)}
              className="group inline-flex items-center gap-1.5 px-2 py-1.5 -mx-2 rounded-md transition-colors hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              aria-label="Lost access? Recover your account"
            >
              <KeyRound className="mr-0.5 -mt-px h-3.5 w-3.5 text-muted-foreground/60 group-hover:text-foreground/80 transition-colors" aria-hidden="true" />
              <span>Lost access? </span>
              <span className="underline underline-offset-4">Recover your account</span>
            </button>
          </div>

          <div className="mt-1 flex items-center justify-center gap-2 text-xs text-muted-foreground">
            <span>Version</span>
            <span className="inline-flex items-center rounded-full border px-2 py-0.5 leading-none">
              {BRAND.version}
            </span>
          </div>
        </section>

        <Dialog open={requestOpen} onOpenChange={(o) => { setRequestOpen(o); if (!o) { setSubmittedEmail(null); navigate("/welcome", { replace: true }); } }}>
          <DialogContent aria-describedby="registration-description" className="max-w-md">
            {submittedEmail ? (
              <div className="space-y-4">
                <DialogHeader>
                  <DialogTitle>Request received</DialogTitle>
                  <DialogDescription id="registration-description">
                    We’ll notify you at {submittedEmail} once your request is approved.
                  </DialogDescription>
                </DialogHeader>
                <div className="flex items-center gap-3 rounded-md border p-3">
                  <CheckCircle2 className="text-primary" aria-hidden="true" />
                  <p className="text-sm">Thanks! Your registration is being reviewed.</p>
                </div>
                <div className="flex justify-end">
                  <Button onClick={() => setRequestOpen(false)}>Close</Button>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                <DialogHeader>
                  <DialogTitle>Request access</DialogTitle>
                  <DialogDescription id="registration-description">
                    Enter your work email to request access.
                  </DialogDescription>
                </DialogHeader>
                <RegisterRequestForm defaultEmail={navState?.email} autoSubmit={navState?.auto === true && requestOpen} onDone={(email) => setSubmittedEmail(email)} />
              </div>
            )}
          </DialogContent>
        </Dialog>

        <Dialog open={recoveryOpen} onOpenChange={(o) => { setRecoveryOpen(o); if (!o) { recForm.reset({ key: "" }); navigate("/welcome", { replace: true }); } }}>
          <DialogContent aria-describedby="recovery-description" className="max-w-md">
            <div className="space-y-4">
              <DialogHeader>
                <DialogTitle>Recover your account</DialogTitle>
                <DialogDescription id="recovery-description">
                  Enter one of your recovery keys to proceed.
                </DialogDescription>
              </DialogHeader>
              <Form {...recForm}>
                <form onSubmit={recForm.handleSubmit(onRecover)} className="space-y-4" noValidate>
                  <FormField
                    control={recForm.control}
                    name="key"
                    render={({ field }) => (
                      <FormItem className="group">
<FormLabel className="flex items-center gap-2">
  <span>Recovery key</span>
  <Popover>
    <PopoverTrigger asChild>
      <button
        type="button"
        className="inline-flex items-center text-muted-foreground hover:text-foreground focus:outline-none"
        aria-label="Recovery key format information"
      >
        <Info className="h-4 w-4" aria-hidden="true" />
      </button>
    </PopoverTrigger>
    <PopoverContent className="w-80">
      <div className="space-y-2 text-sm">
        <p>Use Base32 uppercase A–Z (excluding I, L, O) and digits 2–7.</p>
        <p>Accepted lengths: 20, 25, or 26 characters.</p>
        <p className="font-mono text-xs text-foreground/80">Example: ABCDE-FGHJK-MNPQR-STUVW-XYZ23</p>
      </div>
    </PopoverContent>
  </Popover>
</FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            placeholder="ABCDE-FGHJK-MNPQR-STUVW-XYZ23"
                            inputMode="text"
                            autoCapitalize="characters"
                            autoCorrect="off"
                            spellCheck={false}
                            maxLength={31}
                            className={jiggle ? "animate-jiggle" : undefined}
                            onChange={(e) => {
                              const normalized = normalizeRecoveryKeyInput(e.currentTarget.value);
                              const overshoot = normalized.length > 26;
                              const clamped = normalized.slice(0, 26);
                              const next = formatRecoveryKey(clamped);
                              if (overshoot) {
                                setJiggle(true);
                                window.setTimeout(() => setJiggle(false), 180);
                              }
                              field.onChange(next);
                            }}
                          />
                        </FormControl>
<FormMessage
  role="status"
  aria-live="polite"
  className={cn(
    "mt-1.5 flex items-start gap-2 text-[13px]",
    !fieldError && status?.tone === "success" && "text-success",
    !fieldError && status?.tone === "warning" && "text-warning"
  )}
>
  {fieldError ? (
    <span className="fade-in-up">{fieldError}</span>
  ) : status ? (
    <>
      {status.tone === "success" ? (
        <CheckCircle2 className="h-4 w-4 mt-0.5" aria-hidden="true" />
      ) : (
        <AlertTriangle className="h-4 w-4 mt-0.5" aria-hidden="true" />
      )}
      <span key={`${status.tone}-${rawLen}`} className="fade-in-up">{status.msg}</span>
    </>
  ) : null}
</FormMessage>
                        </FormItem>
                    )}
                  />
                  <div className="flex justify-end gap-2">
                    <Button type="button" variant="outline" onClick={() => setRecoveryOpen(false)}>
                      Cancel
                    </Button>
                    <Button type="submit" disabled={recForm.formState.isSubmitting}>
                      {recForm.formState.isSubmitting ? "Verifying..." : "Continue"}
                    </Button>
                  </div>
                </form>
              </Form>
            </div>
          </DialogContent>
        </Dialog>

      </main>
    </div>
  );
}
