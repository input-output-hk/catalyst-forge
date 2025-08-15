import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAppStore } from "@/store/app-store";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import { LogoMark } from "@/components/brand/LogoMark";

const Step = ({ title, children }: { title: string; children: React.ReactNode }) => (
  <Card className="mb-4">
    <CardHeader><CardTitle>{title}</CardTitle></CardHeader>
    <CardContent>{children}</CardContent>
  </Card>
);

const AuthFlows = () => {
  const { actions } = useAppStore();
  const [message, setMessage] = useState<string>("");
  const nav = useNavigate();
  const helmet = usePageTitle("Auth Flows – Catalyst Forge", "Click-through demo of auth scenarios.", "/auth-demo");

  return (
    <section className="container py-8">
      {helmet}
      <div className="mb-6 rounded-lg border bg-hero p-6">
        <div className="flex items-center gap-3">
          <LogoMark size={40} />
          <div>
            <div className="text-xs uppercase tracking-widest text-primary font-semibold">Catalyst Forge</div>
            <div className="text-sm text-muted-foreground">Authentication demo</div>
          </div>
        </div>
      </div>
      <h1 className="text-2xl font-bold mb-4">Auth Flows</h1>
      <Step title="Brand‑new user onboarding">
        <Button variant="hero" onClick={() => setMessage("Account created → email verified → WebAuthn credential registered → landing on Dashboard")}>Start</Button>
      </Step>
      <Step title="Add new credential">
        <Button onClick={() => setMessage("Prompted for device → credential added")}>Add credential</Button>
      </Step>
      <Step title="Normal session">
        <Button variant="secondary" onClick={() => setMessage("Session valid; navigating around shows authed state")}>Show</Button>
      </Step>
      <Step title="Access‑token refresh">
        <Button variant="outline" onClick={() => setMessage("Token expired → silent refresh → last action retried")}>Simulate</Button>
      </Step>
      <Step title="Returning login (WebAuthn)">
        <Button onClick={() => setMessage("Sign‑in with passkey → success")}>Sign in</Button>
      </Step>
      <Step title="Account recovery">
        <Button variant="outline" onClick={() => setMessage("Identity proofed → new credential registered")}>Recover</Button>
      </Step>
      <Step title="Forced re‑login">
        <Button variant="destructive" onClick={() => setMessage("Session invalidated → CTA to sign in again")}>Force</Button>
      </Step>
      <Step title="Logout / Logout all">
        <div className="flex gap-2">
          <Button onClick={() => { actions.logout(); setMessage("Logged out on this device"); }}>Logout</Button>
          <Button variant="outline" onClick={() => setMessage("All sessions cleared")}>Logout all</Button>
        </div>
      </Step>
      {message && <p className="text-sm text-muted-foreground mt-2">{message}</p>}
    </section>
  );
};

export default AuthFlows;
