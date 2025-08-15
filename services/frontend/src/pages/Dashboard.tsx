import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/store/app-store";
import { Link } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import { BRAND } from "@/lib/brand";

const Stat = ({ label, value, to }: { label: string; value: string | number; to: string }) => (
  <Link to={to} className="block hover-scale">
    <Card className="surface-card">
      <CardHeader>
        <CardTitle>{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-semibold">{value}</div>
      </CardContent>
    </Card>
  </Link>
);

const Dashboard = () => {
  const { state } = useAppStore();
  const helmet = usePageTitle("Dashboard – Catalyst Forge", "Overview of services, jobs, and activity.", "/");

  return (
    <div className="bg-hero">
      {helmet}
      <section className="container mx-auto py-6">
        <div className="mb-6">
          <div className="text-[11px] uppercase tracking-widest text-primary font-semibold flex items-center gap-2">
            <span>{BRAND.name}</span>
            <span className="inline-flex items-center rounded-full border border-primary/50 text-primary px-2 py-0.5 text-[10px]">{BRAND.version}</span>
          </div>
          <h1 className="text-3xl font-extrabold text-gradient">Welcome back</h1>
          <span className="forge-arc mt-2" aria-hidden />
          <p className="text-muted-foreground">Your platform at a glance</p>
        </div>
        <div className="grid md:grid-cols-3 gap-4">
          <Stat label="Services" value={state.services.length} to="/services" />
          <Stat label="Environments" value={state.environments.length} to="/environments" />
          <Stat label="Recent Jobs" value={state.jobs.length} to="/jobs" />
        </div>
        <div className="mt-8">
          <a className={cn(buttonVariants({ variant: "hero" }), "shadow-glow")} href="/auth-demo">Explore Auth Flows</a>
        </div>
      </section>
    </div>
  );
};

export default Dashboard;
