import { useEffect, useState } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { LogoMark } from "@/components/brand/LogoMark";
import { ForgeAnimation } from "@/components/brand/ForgeAnimation";
import { BRAND } from "@/lib/brand";
import { preloadAuthFlows } from "@/lib/preloaders";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { CheckCircle2, KeyRound, Info, AlertTriangle } from "lucide-react";
import RegisterRequestForm from "@/components/auth/RegisterRequestForm";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { useToast } from "@/hooks/use-toast";
import { useAppStore } from "@/store/app-store";
import { loginWithWebAuthn } from "@/lib/webauthn";
import {
  formatRecoveryKey,
  isValidRecoveryKey,
  normalizeRecoveryKeyInput,
} from "@/lib/auth/recovery";
import { cn } from "@/lib/utils";
import { usePageTitle } from "@/hooks/usePageTitle";
import { extractErrorMessage } from "@/lib/api";

// Type definitions
interface NavigationState {
  email?: string;
  auto?: boolean;
  from?: string;
}

interface RecoveryKeyStatus {
  tone: "success" | "warning";
  msg: string;
}

type RecoveryFormData = z.infer<typeof recoverySchema>;

// Schema definitions
const recoverySchema = z.object({
  key: z
    .string()
    .min(1, "Recovery key is required")
    .refine((value) => isValidRecoveryKey(value), {
      message: "Use A–Z and 2–7; length 20, 25, or 26.",
    }),
});
/**
 * Landing page component for Catalyst Forge.
 * Handles user authentication, access requests, and account recovery.
 */
export default function Landing() {
  // Navigation and routing
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();

  // Extract navigation state
  const navigationState = location.state as NavigationState | null;
  const shouldOpenRequest = searchParams.get("request") === "1";
  const shouldOpenRecovery = searchParams.get("recover") === "1";

  // Dialog state management
  const [requestOpen, setRequestOpen] = useState(false);
  const [recoveryOpen, setRecoveryOpen] = useState(false);
  const [submittedEmail, setSubmittedEmail] = useState<string | null>(null);

  // UI state
  const [jiggle, setJiggle] = useState(false);

  // Open dialogs based on URL parameters
  useEffect(() => {
    if (shouldOpenRequest) {
      setRequestOpen(true);
    }
  }, [shouldOpenRequest]);

  useEffect(() => {
    if (shouldOpenRecovery) {
      setRecoveryOpen(true);
    }
  }, [shouldOpenRecovery]);

  // Recovery form setup
  const recoveryForm = useForm<RecoveryFormData>({
    resolver: zodResolver(recoverySchema),
    defaultValues: { key: "" },
  });

  // Auto-focus recovery key field when dialog opens
  useEffect(() => {
    if (recoveryOpen) {
      const focusDelay = 10; // milliseconds
      setTimeout(() => recoveryForm.setFocus("key"), focusDelay);
    }
  }, [recoveryOpen, recoveryForm]);

  // Store and hooks
  const { actions } = useAppStore();
  const { toast } = useToast();

  // SEO metadata
  const seo = usePageTitle(
    "Catalyst Forge – Developer Platform",
    "Developer platform landing: login, access request, and account recovery.",
    "/welcome"
  );

  /**
   * Handles WebAuthn login flow.
   * Shows authentication prompt and navigates on success.
   */
  async function handleLogin() {
    try {
      toast({
        title: "Authenticate",
        description: "Touch your security key or biometric sensor.",
      });

      await loginWithWebAuthn();

      // Navigate to previous page or home
      const redirectPath = navigationState?.from || "/";
      navigate(redirectPath, { replace: true });
    } catch (error) {
      const errorMessage = extractErrorMessage(error, "Login failed");
      toast({
        title: "Login failed",
        description: errorMessage,
        variant: "destructive",
      });
    }
  }

  /**
   * Handles account recovery form submission.
   * Verifies recovery key and registers current device.
   */
  async function handleRecovery(_values: RecoveryFormData) {
    // Log recovery attempt
    actions.addAudit({
      actor: "recovery",
      action: "recovery.verify",
      resource: "account",
      meta: { method: "key" },
    });

    // Show success message
    toast({
      title: "Recovery key accepted",
      description: "Registering this device...",
    });

    // Close dialog and reset form
    setRecoveryOpen(false);
    recoveryForm.reset({ key: "" });

    // Navigate to welcome page
    navigate("/welcome", { replace: true });

    // Register device asynchronously
    await actions.addDevice("This device");
  }

  /**
   * Calculates recovery key validation status based on length.
   * Provides real-time feedback as user types.
   */
  function getRecoveryKeyStatus(keyValue: string): RecoveryKeyStatus | null {
    const normalizedKey = normalizeRecoveryKeyInput(keyValue || "");
    const keyLength = normalizedKey.length;
    const validLengths = [20, 25, 26] as const;
    const isValidLength = validLengths.includes(keyLength as (typeof validLengths)[number]);

    if (keyLength === 0) {
      return null;
    }

    if (keyLength < 20) {
      const charactersNeeded = 20 - keyLength;
      return {
        tone: "warning",
        msg: `Almost there — ${charactersNeeded} more characters needed (min 20).`,
      };
    }

    if (isValidLength) {
      return {
        tone: "success",
        msg: "Key format is valid.",
      };
    }

    return {
      tone: "warning",
      msg: `Accepted lengths: 20, 25, or 26. Currently ${keyLength}.`,
    };
  }

  // Watch recovery key field for real-time validation
  const recoveryKeyValue = recoveryForm.watch("key");
  const recoveryKeyStatus = getRecoveryKeyStatus(recoveryKeyValue);
  const recoveryKeyError = recoveryForm.formState.errors.key?.message;

  return (
    <div className="relative min-h-screen bg-background text-foreground overflow-hidden">
      {seo}

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
              onClick={handleLogin}
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
              <KeyRound
                className="mr-0.5 -mt-px h-3.5 w-3.5 text-muted-foreground/60 group-hover:text-foreground/80 transition-colors"
                aria-hidden="true"
              />
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

        {/* Access Request Dialog */}
        <Dialog
          open={requestOpen}
          onOpenChange={(isOpen) => {
            setRequestOpen(isOpen);

            if (!isOpen) {
              setSubmittedEmail(null);
              navigate("/welcome", { replace: true });
            }
          }}
        >
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
                <RegisterRequestForm
                  defaultEmail={navigationState?.email}
                  autoSubmit={navigationState?.auto === true && requestOpen}
                  onDone={(email) => setSubmittedEmail(email)}
                />
              </div>
            )}
          </DialogContent>
        </Dialog>

        {/* Account Recovery Dialog */}
        <Dialog
          open={recoveryOpen}
          onOpenChange={(isOpen) => {
            setRecoveryOpen(isOpen);

            if (!isOpen) {
              recoveryForm.reset({ key: "" });
              navigate("/welcome", { replace: true });
            }
          }}
        >
          <DialogContent aria-describedby="recovery-description" className="max-w-md">
            <div className="space-y-4">
              <DialogHeader>
                <DialogTitle>Recover your account</DialogTitle>
                <DialogDescription id="recovery-description">
                  Enter one of your recovery keys to proceed.
                </DialogDescription>
              </DialogHeader>
              <Form {...recoveryForm}>
                <form
                  onSubmit={recoveryForm.handleSubmit(handleRecovery)}
                  className="space-y-4"
                  noValidate
                >
                  <FormField
                    control={recoveryForm.control}
                    name="key"
                    render={({ field }) => (
                      <FormItem className="group">
                        <FormLabel className="flex items-center gap-2">
                          <span>Recovery key</span>

                          {/* Recovery key format help */}
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
                                <p className="font-mono text-xs text-foreground/80">
                                  Example: ABCDE-FGHJK-MNPQR-STUVW-XYZ23
                                </p>
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
                            onChange={(event) => {
                              const inputValue = event.currentTarget.value;
                              const normalizedValue = normalizeRecoveryKeyInput(inputValue);

                              // Check if user exceeded max length
                              const hasExceededMaxLength = normalizedValue.length > 26;
                              const clampedValue = normalizedValue.slice(0, 26);
                              const formattedValue = formatRecoveryKey(clampedValue);

                              // Trigger jiggle animation if exceeded
                              if (hasExceededMaxLength) {
                                setJiggle(true);
                                const animationDuration = 180; // milliseconds
                                window.setTimeout(() => setJiggle(false), animationDuration);
                              }

                              field.onChange(formattedValue);
                            }}
                          />
                        </FormControl>
                        {/* Validation feedback */}
                        <FormMessage
                          role="status"
                          aria-live="polite"
                          className={cn(
                            "mt-1.5 flex items-start gap-2 text-[13px]",
                            !recoveryKeyError &&
                              recoveryKeyStatus?.tone === "success" &&
                              "text-success",
                            !recoveryKeyError &&
                              recoveryKeyStatus?.tone === "warning" &&
                              "text-warning"
                          )}
                        >
                          {recoveryKeyError ? (
                            <span className="fade-in-up">{recoveryKeyError}</span>
                          ) : recoveryKeyStatus ? (
                            <>
                              {recoveryKeyStatus.tone === "success" ? (
                                <CheckCircle2 className="h-4 w-4 mt-0.5" aria-hidden="true" />
                              ) : (
                                <AlertTriangle className="h-4 w-4 mt-0.5" aria-hidden="true" />
                              )}
                              <span
                                key={`${recoveryKeyStatus.tone}-${recoveryKeyValue.length}`}
                                className="fade-in-up"
                              >
                                {recoveryKeyStatus.msg}
                              </span>
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
                    <Button type="submit" disabled={recoveryForm.formState.isSubmitting}>
                      {recoveryForm.formState.isSubmitting ? "Verifying..." : "Continue"}
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
