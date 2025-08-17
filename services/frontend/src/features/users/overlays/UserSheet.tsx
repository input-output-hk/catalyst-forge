import { useEffect, useState } from "react";
import { format, formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Key, MoreVertical, Copy, LogOut } from "lucide-react";
import { cn } from "@/lib/utils";
import { useToast } from "@/hooks/use-toast";
import { forgeFetch } from "@/lib/client";
import {
  adminGenerateUserRecoveryCodes,
  adminListAudit,
  adminListUserCredentials,
} from "@/lib/auth/admin";
import { useAppStore } from "@/store/app-store";
import type { User } from "@/features/users/types";

function firstLast(name: string) {
  const parts = name.split(" ");
  const first = parts[0]?.[0] ?? "?";
  const last = parts[1]?.[0] ?? parts[0]?.[1] ?? "";
  return (first + last).toUpperCase();
}

function StatusBadge({ status }: { status: User["status"] }) {
  // Minimal duplication; ideally import from a shared component if/when extracted
  const map = {
    active: { label: "Active", tooltip: "Active" },
    disabled: { label: "Disabled", tooltip: "Disabled account" },
    pending_invite: { label: "Pending", tooltip: "Invite sent, awaiting acceptance" },
    pending_approval: { label: "Pending", tooltip: "Awaiting admin approval" },
  } as const;
  const { label, tooltip } = map[status];
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex items-center justify-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 min-w-[96px] bg-muted text-muted-foreground ring-border/30">
          <span>{label}</span>
        </span>
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}

function RoleBadge({ role }: { role: User["role"] }) {
  return (
    <span className="rounded-full border border-muted-foreground/60 px-2 py-0.5 text-xs text-muted-foreground">
      {role === "admin" ? "Admin" : "Member"}
    </span>
  );
}

type APICred = { id: string; device_name: string; sign_count: number; last_used_at?: string };

type Props = {
  user: User;
  onOpenChange: (open: boolean) => void;
  density: "comfortable" | "compact";
};

export default function UserSheet({ user, onOpenChange, density }: Props) {
  const { toast } = useToast();
  const { state } = useAppStore();
  const currentUserEmail = state.session.user || "";

  const [openUserCreds, setOpenUserCreds] = useState<APICred[]>([]);
  const [loadingOpenCreds, setLoadingOpenCreds] = useState(false);
  const [openUserAudit, setOpenUserAudit] = useState<
    Array<{ id: string; type: string; actor_id?: string; created_at: string }>
  >([]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        setLoadingOpenCreds(true);
        const creds = await adminListUserCredentials(user.id);
        if (!cancelled) setOpenUserCreds(creds.credentials || []);
        const aj = await adminListAudit({ user_id: user.id, limit: 10 });
        const simplified = (aj.events || []).map((e) => ({
          id: e.id,
          type: e.type,
          actor_id: e.actor_id,
          created_at: e.created_at,
        }));
        if (!cancelled) setOpenUserAudit(simplified);
      } catch {
        if (!cancelled) {
          setOpenUserCreds([]);
          setOpenUserAudit([]);
        }
      } finally {
        if (!cancelled) setLoadingOpenCreds(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [user.id]);

  return (
    <Sheet open={!!user} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-full sm:max-w-[520px] md:max-w-[560px] motion-safe:animate-in fade-in slide-in-from-right-2"
      >
        <div className="flex h-full flex-col">
          <SheetHeader className="sticky top-0 z-10 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <SheetTitle>
              <div className="rounded-lg border border-white/5 bg-foreground/[0.02] shadow-inner">
                <div
                  className={cn(
                    "grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3",
                    density === "compact" ? "px-4 pt-4 pb-3" : "px-5 pt-5 pb-4"
                  )}
                >
                  <div className="min-w-0 flex items-center gap-3">
                    <Avatar className="h-9 w-9">
                      <AvatarFallback>{firstLast(user.name)}</AvatarFallback>
                    </Avatar>
                    <div className="min-w-0">
                      <div className="truncate font-semibold leading-tight">
                        {user.name && user.name.trim().toLowerCase() !== user.email.toLowerCase()
                          ? user.name
                          : user.email}
                      </div>
                      {user.name && user.name.trim().toLowerCase() !== user.email.toLowerCase() && (
                        <div className="truncate text-xs text-muted-foreground">{user.email}</div>
                      )}
                      <div className="mt-1 flex flex-wrap items-center gap-2">
                        <RoleBadge role={user.role} />
                        <StatusBadge status={user.status} />
                      </div>
                    </div>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button size="icon" variant="ghost" className="h-8 w-8 shrink-0">
                        <MoreVertical className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      {user.status === "disabled" ? (
                        <DropdownMenuItem
                          onClick={() => {
                            // The parent is expected to adjust user state; this is a light overlay.
                            // Emitting an event keeps coupling low.
                            window.dispatchEvent(
                              new CustomEvent("cf:user-toggle-enabled", {
                                detail: { id: user.id, enable: true },
                              })
                            );
                          }}
                        >
                          Enable
                        </DropdownMenuItem>
                      ) : (
                        <DropdownMenuItem
                          disabled={user.email === currentUserEmail || user.role === "admin"}
                          title={
                            user.email === currentUserEmail
                              ? "You can’t disable yourself"
                              : user.role === "admin"
                                ? "Admins cannot be disabled here"
                                : undefined
                          }
                          onClick={async () => {
                            if (user.email === currentUserEmail || user.role === "admin") return;
                            try {
                              const res = await forgeFetch(`/api/v1/admin/users/${user.id}`, {
                                method: "PATCH",
                                headers: { "content-type": "application/json" },
                                body: JSON.stringify({ suspend: true }),
                              });
                              if (res.status === 204) {
                                window.dispatchEvent(
                                  new CustomEvent("cf:user-toggled", {
                                    detail: { id: user.id, disabled: true },
                                  })
                                );
                              } else {
                                const text = await res.text().catch(() => "");
                                toast({ title: "Failed", description: text || `${res.status}` });
                              }
                            } catch {
                              toast({ title: "Network error" });
                            }
                          }}
                        >
                          Disable
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuItem
                        onClick={() => {
                          window.dispatchEvent(
                            new CustomEvent("cf:user-logout", { detail: { id: user.id } })
                          );
                        }}
                      >
                        Log out
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </SheetTitle>
          </SheetHeader>

          <div className="flex-1 overflow-y-auto">
            <section
              className={cn(
                "border-b border-white/[0.06]",
                density === "compact" ? "px-4 py-3" : "px-5 py-4"
              )}
            >
              <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Overview
              </h3>
              <div className="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
                <div>
                  <div className="text-muted-foreground">User ID</div>
                  <div className="flex items-center gap-2 font-mono text-xs break-all">
                    <span className="truncate">{user.id}</span>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            size="icon"
                            variant="ghost"
                            aria-label="Copy user ID"
                            onClick={async () => {
                              try {
                                await navigator.clipboard.writeText(user.id);
                                toast({ title: "Copied user ID" });
                              } catch {
                                // ignore
                              }
                            }}
                          >
                            <Copy className="h-4 w-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Copy</TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Role</div>
                  <div>
                    <RoleBadge role={user.role} />
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Created</div>
                  <div>{format(new Date(user.createdAt), "PPpp")}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">Active sessions</div>
                  <div>{user.sessions}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">Last login</div>
                  <div>
                    {user.lastActivityAt ? format(new Date(user.lastActivityAt), "PPpp") : "Never"}
                  </div>
                </div>
              </div>
            </section>

            <section
              className={cn(
                "border-b border-white/[0.06]",
                density === "compact" ? "px-4 py-3" : "px-5 py-4"
              )}
            >
              <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Credentials
              </h3>
              <div className="mt-3 space-y-2">
                {loadingOpenCreds ? (
                  <div className="text-sm text-muted-foreground">Loading…</div>
                ) : openUserCreds.length === 0 ? (
                  <div className="text-sm text-muted-foreground">No credentials added.</div>
                ) : (
                  openUserCreds
                    .slice()
                    .sort((a, b) => {
                      const ta = a.last_used_at ? new Date(a.last_used_at).getTime() : 0;
                      const tb = b.last_used_at ? new Date(b.last_used_at).getTime() : 0;
                      return tb - ta;
                    })
                    .map((c) => (
                      <article
                        key={c.id}
                        className="flex items-center justify-between rounded-md border bg-background/[0.6] px-3 py-2"
                      >
                        <div className="flex items-center gap-3">
                          <Key className="h-4 w-4 text-muted-foreground" />
                          <div className="text-sm">
                            <div className="font-medium">{c.device_name}</div>
                            <div className="text-xs text-muted-foreground">
                              Sign count {c.sign_count} • Last used{" "}
                              {c.last_used_at ? format(new Date(c.last_used_at), "PP") : "—"}
                            </div>
                          </div>
                        </div>
                      </article>
                    ))
                )}
              </div>
            </section>

            <section
              className={cn(
                "border-b border-white/[0.06]",
                density === "compact" ? "px-4 py-3" : "px-5 py-4"
              )}
            >
              <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Security
              </h3>
              <div className="mt-3 flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                  Recovery keys: Generate new one-time codes for this user.
                </div>
                <Button
                  size="sm"
                  onClick={async () => {
                    try {
                      const data = await adminGenerateUserRecoveryCodes(user.id);
                      const filename = `recovery_codes_${user.email}_${format(new Date(), "yyyyMMdd_HHmmss")}.txt`;
                      const { downloadRecoveryCodes } = await import("@/lib/auth/recovery");
                      downloadRecoveryCodes(filename, data.codes);
                      toast({
                        title: "Recovery keys generated",
                        description: "A .txt file was downloaded with the one-time codes.",
                      });
                    } catch {
                      toast({ title: "Failed to generate recovery keys" });
                    }
                  }}
                >
                  Regenerate
                </Button>
              </div>
            </section>

            <section className={cn(density === "compact" ? "px-4 py-3" : "px-5 py-4")}>
              <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Audit
              </h3>
              <div className="mt-3">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-xs text-muted-foreground">
                        <th className="text-left font-normal">Actor</th>
                        <th className="text-left font-normal">Type</th>
                        <th className="text-left font-normal">Date</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(openUserAudit || []).slice(0, 4).map((evt) => (
                        <tr key={evt.id} className="border-t border-white/[0.06]">
                          <td className="py-2">{evt.actor_id || ""}</td>
                          <td className="py-2">{evt.type}</td>
                          <td className="py-2">{format(new Date(evt.created_at), "PP")}</td>
                        </tr>
                      ))}
                      {(!openUserAudit || openUserAudit.length === 0) && (
                        <tr>
                          <td className="py-2 text-muted-foreground" colSpan={3}>
                            No recent events
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
                <div className="mt-2 text-right">
                  <a href="/audit" className="text-sm story-link">
                    Open in Audit Log
                  </a>
                </div>
              </div>
            </section>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
