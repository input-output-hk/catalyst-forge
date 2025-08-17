import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Skeleton } from "@/components/ui/skeleton";
import { can, EnvHealth } from "@/lib/mock/dashboard";
import { Link } from "react-router-dom";
import { Lock, LogIn, Rocket } from "lucide-react";

function formatAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  return `${hrs}h ago`;
}

function ActionButton({ env }: { env: EnvHealth["id"] }) {
  const actionKey =
    env === "prod" ? "promote:prod" : env === "preprod" ? "promote:preprod" : "promote:dev";
  const perm = can(actionKey);
  const btn = (
    <Button size="sm" variant="secondary" disabled={!perm.allowed} aria-label={`Promote ${env}`}>
      {!perm.allowed ? <Lock className="mr-2" /> : <Rocket className="mr-2" />} Promote
    </Button>
  );
  if (perm.allowed) return btn;
  return (
    <Tooltip>
      <TooltipTrigger asChild>{btn}</TooltipTrigger>
      <TooltipContent>{perm.reason}</TooltipContent>
    </Tooltip>
  );
}

export function EnvironmentHealthSkeleton() {
  return (
    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <Card key={i} className="surface-card">
          <CardHeader>
            <Skeleton className="h-5 w-28" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between mb-2">
              <Skeleton className="h-6 w-20" />
              <Skeleton className="h-5 w-16" />
            </div>
            <div className="flex items-center gap-2">
              <Skeleton className="h-9 w-24" />
              <Skeleton className="h-9 w-24" />
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

export default function EnvironmentHealth({ envs }: { envs: EnvHealth[] }) {
  return (
    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
      {envs.map((e) => (
        <Card key={e.id} className="surface-card">
          <CardHeader>
            <CardTitle className="text-base capitalize">{e.id}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="font-medium">{e.release}</div>
              <div className="text-sm text-muted-foreground">{formatAgo(e.lastDeployAt)}</div>
            </div>
            <div className="mt-2 flex items-center gap-2">
              <Badge variant={e.drift ? "destructive" : "secondary"}>
                {e.drift ? "Drift" : "In sync"}
              </Badge>
              <Badge variant={e.errorRate > 1 ? "destructive" : "secondary"}>
                {e.errorRate.toFixed(1)}% errors
              </Badge>
            </div>
            <div className="mt-3 flex items-center gap-2">
              <ActionButton env={e.id} />
              <Button asChild size="sm" variant="outline" aria-label={`Open logs for ${e.id}`}>
                <Link to="/jobs">
                  <LogIn className="mr-2" /> Open logs
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
