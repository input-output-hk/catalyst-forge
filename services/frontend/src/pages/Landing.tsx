import { useEffect, useState } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { LogoMark } from "@/components/brand/LogoMark";
import { ForgeAnimation } from "@/components/brand/ForgeAnimation";
import { BRAND } from "@/lib/brand";
import { Loader2 } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { useAppStore } from "@/store/app-store";
import { loginWithProvider, registerWithProvider } from "@/lib/auth/oidc";
import { usePageTitle } from "@/hooks/usePageTitle";
import { extractErrorMessage } from "@/lib/api";

// Type definitions
interface NavigationState {
  email?: string;
  auto?: boolean;
  from?: string;
}

// Recovery flows removed
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

  // UI state
  const [loginPending, setLoginPending] = useState(false);
  const [registerPending, setRegisterPending] = useState(false);

  // Store and hooks
  const { state, actions } = useAppStore();
  const { toast } = useToast();

  // SEO metadata
  const seo = usePageTitle(
    "Catalyst Forge – Developer Platform",
    "Developer platform landing: login, register, and account recovery.",
    "/welcome"
  );

  // If a valid Kratos session already exists (e.g., right after registration),
  // mark the app as authed and send the user to their intended destination.
  useEffect(() => {
    let cancelled = false;
    async function pickUpExistingSession() {
      try {
        if (!state.session.authed) {
          const resp = await fetch("/.ory/kratos/public/sessions/whoami", { credentials: "include" });
          if (resp.ok) {
            const data = await resp.json();
            const email = data?.identity?.traits?.email ?? "user";
            actions.login(email, Array.isArray(state.session.roles) ? state.session.roles : []);
            const redirectPath = navigationState?.from || "/";
            navigate(redirectPath, { replace: true });
          }
        }
      } catch {
        // no-op; stay on welcome
      }
    }
    pickUpExistingSession();
    return () => {
      cancelled = true;
    };
  }, [state.session.authed, state.session.roles, navigationState?.from, actions, navigate]);

  async function handleLogin() {
    const redirectPath = navigationState?.from || "/";
    try {
      setLoginPending(true);
      await loginWithProvider("google", redirectPath);
    } catch (err) {
      setLoginPending(false);
      toast({
        title: "Login failed",
        description: extractErrorMessage(err, "Could not start sign-in. Please try again."),
        variant: "destructive",
      });
    }
  }

  async function handleRegister() {
    const redirectPath = navigationState?.from || "/";
    try {
      setRegisterPending(true);
      await registerWithProvider("google", redirectPath);
    } catch (err) {
      setRegisterPending(false);
      toast({
        title: "Registration failed",
        description: extractErrorMessage(err, "Could not start registration. Please try again."),
        variant: "destructive",
      });
    }
  }

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

          <div className="mt-8 grid grid-cols-1 gap-3 md:gap-4">
            <Button
              size="lg"
              variant="hero"
              className="btn-ripple hover-scale hover-glow w-full"
              aria-label="Login"

              onClick={handleLogin}
              disabled={loginPending}
              aria-disabled={loginPending}
            >
              {loginPending ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" /> Redirecting…
                </span>
              ) : (
                "Login"
              )}
            </Button>
            <Button
              size="lg"
              variant="secondary"
              className="btn-ripple hover-scale hover-glow w-full"
              onClick={handleRegister}
              disabled={registerPending}
              aria-disabled={registerPending}
            >
              {registerPending ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" /> Redirecting…
                </span>
              ) : (
                "Create account"
              )}
            </Button>
          </div>


          <div className="mt-1 flex items-center justify-center gap-2 text-xs text-muted-foreground">
            <span>Version</span>
            <span className="inline-flex items-center rounded-full border px-2 py-0.5 leading-none">
              {BRAND.version}
            </span>
          </div>
        </section>
      </main>
    </div>
  );
}
