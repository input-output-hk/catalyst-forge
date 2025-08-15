import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { ActivityItem } from "@/lib/mock/dashboard";
import { Link } from "react-router-dom";

function msToReadable(ms: number) {
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  const sec = s % 60;
  return `${m}m ${sec}s`;
}

function elapsed(startedAt: string, finishedAt?: string) {
  const end = finishedAt ? new Date(finishedAt) : new Date();
  const diff = end.getTime() - new Date(startedAt).getTime();
  return msToReadable(diff);
}

const StatusPill = ({ status }: { status: ActivityItem["status"] }) => {
  const variant = status === "failed" ? "destructive" : status === "queued" ? "secondary" : status === "running" ? "secondary" : "default";
  return <Badge variant={variant as any} className="capitalize">{status}</Badge>;
};

export function ActivityFeedSkeleton() {
  return (
    <Card className="surface-card">
      <CardHeader>
        <CardTitle className="text-base">Live Activity</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Skeleton className="h-5 w-16" />
                <Skeleton className="h-4 w-48" />
                <Skeleton className="h-5 w-14" />
              </div>
              <Skeleton className="h-8 w-32" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

export default function ActivityFeed({ items }: { items: ActivityItem[] }) {
  return (
    <Card className="surface-card">
      <CardHeader>
        <CardTitle className="text-base">Live Activity</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {items.map((a) => {
            const progress = a.status === "running" ? Math.min(100, Math.max(5, ((Date.now() - new Date(a.startedAt).getTime()) / (6 * 60 * 1000)) * 100)) : a.status === "queued" ? 0 : 100;
            return (
              <article key={a.id} className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3 min-w-0">
                  <StatusPill status={a.status} />
                  <div className="truncate">
                    <div className="font-medium truncate">{a.name}</div>
                    <div className="text-sm text-muted-foreground">{a.env} • {elapsed(a.startedAt, a.finishedAt)}</div>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {a.status === "running" && (
                    <div className="w-40 hidden md:block"><Progress value={progress} aria-label="Progress" /></div>
                  )}
                  <Button asChild size="sm" variant="outline" aria-label="View logs">
                    <Link to="/jobs">View logs</Link>
                  </Button>
                  <Button size="sm" variant="ghost" aria-label="Retry" disabled={a.status === "running" || a.status === "queued"}>Retry</Button>
                </div>
              </article>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
