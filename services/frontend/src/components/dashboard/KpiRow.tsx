import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Kpi } from "@/lib/mock/dashboard";
import { TrendingUp, Timer, GitBranch, ShieldAlert, ShieldCheck } from "lucide-react";
import { ResponsiveContainer, AreaChart, Area } from "recharts";

const iconFor = (id: string) => {
  switch (id) {
    case "deploy_success":
      return <TrendingUp className="text-muted-foreground" aria-hidden />;
    case "build_queue":
      return <Timer className="text-muted-foreground" aria-hidden />;
    case "preview_envs":
      return <GitBranch className="text-muted-foreground" aria-hidden />;
    case "secrets_overdue":
      return <ShieldAlert className="text-muted-foreground" aria-hidden />;
    case "certs_expiring":
      return <ShieldCheck className="text-muted-foreground" aria-hidden />;
    default:
      return <TrendingUp className="text-muted-foreground" aria-hidden />;
  }
};

const StatusBadge = ({ kind }: { kind: "good" | "warn" | "bad" }) => {
  const variant = kind === "bad" ? "destructive" : kind === "warn" ? "secondary" : "default";
  const label = kind === "bad" ? "At risk" : kind === "warn" ? "Watch" : "Healthy";
  return (
    <Badge variant={variant as "default" | "secondary" | "destructive" | "outline"}>{label}</Badge>
  );
};

export function KpiSkeletonRow() {
  return (
    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
      {Array.from({ length: 5 }).map((_, i) => (
        <Card key={i} className="surface-card">
          <CardHeader className="flex-row items-center justify-between">
            <div className="flex items-center gap-2">
              <Skeleton className="h-5 w-5 rounded" />
              <Skeleton className="h-4 w-32" />
            </div>
            <Skeleton className="h-5 w-16" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-24 mb-2" />
            <Skeleton className="h-10 w-full" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

export default function KpiRow({ kpis }: { kpis: Kpi[] }) {
  return (
    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
      {kpis.map((k) => (
        <Card key={k.id} className="surface-card">
          <CardHeader className="flex-row items-center justify-between">
            <div className="flex items-center gap-2">
              {iconFor(k.id)}
              <CardTitle className="text-base font-semibold">{k.label}</CardTitle>
            </div>
            <StatusBadge kind={k.badge} />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold leading-tight">
              {k.value}
              {k.unit ? (
                <span className="text-muted-foreground text-lg align-top ml-1">{k.unit}</span>
              ) : null}
            </div>
            <div className="mt-2 h-10">
              <ResponsiveContainer width="100%" height={40}>
                <AreaChart data={k.spark.map((v, idx) => ({ idx, v }))}>
                  <Area
                    type="monotone"
                    dataKey="v"
                    stroke="hsl(var(--primary))"
                    fill="hsl(var(--primary) / 0.2)"
                    strokeWidth={2}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
