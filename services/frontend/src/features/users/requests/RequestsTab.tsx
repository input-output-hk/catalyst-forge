import { useEffect, useMemo, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { Search, Users as UsersIcon } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

import { cn } from "@/lib/utils";
import { useToast } from "@/hooks/use-toast";
import EmptyState from "@/components/EmptyState";
import { adminDecideAccessRequest, adminListAccessRequests } from "@/lib/auth/admin";
import type { User } from "@/features/users/types";

type RequestsTabProps = {
  users: User[];
  setUsers: React.Dispatch<React.SetStateAction<User[]>>;
  density: "comfortable" | "compact";
  setDensity: (d: "comfortable" | "compact") => void;
  searchRef: React.RefObject<HTMLInputElement>;
};

type AccessRequest = {
  id: string;
  email: string;
  reason?: string;
  status: string;
  attempts: number;
  decided_at?: string | null;
  created_at: string;
};

export default function RequestsTab({
  users,
  setUsers,
  density,
  setDensity,
  searchRef,
}: RequestsTabProps) {
  const { toast } = useToast();
  const [q, setQ] = useState("");
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectTarget, setRejectTarget] = useState<AccessRequest | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [notify, setNotify] = useState(true);
  const [requests, setRequests] = useState<AccessRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const pageSize = 25;
  const [total, setTotal] = useState<number | null>(null);

  async function load() {
    setLoading(true);
    try {
      const data = await adminListAccessRequests({
        status: "pending",
        q: q.trim() || undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      });
      setRequests((data.requests as unknown as AccessRequest[]) || []);
      setTotal(typeof data.total === "number" ? data.total : null);
      setSelected({});
    } catch (err) {
      toast({ title: "Failed to load access requests", variant: "destructive" });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, page]);

  // Open the parent-level invite dialog with prefilled email, default role "member"
  const approveOne = (r: AccessRequest) => {
    window.dispatchEvent(
      new CustomEvent("cf:open-invite", { detail: { email: r.email, role: "member" } })
    );
  };

  const rejectOne = async (r: AccessRequest, reason?: string) => {
    try {
      await adminDecideAccessRequest(r.id, { approve: false, note: reason || "" });
      toast({ title: `Request rejected for ${r.email}.` });
    } catch {
      toast({ title: `Failed to reject ${r.email}`, variant: "destructive" });
    } finally {
      load();
    }
  };

  const selectedIds = useMemo(() => Object.keys(selected).filter((id) => selected[id]), [selected]);

  return (
    <div>
      <div className="mb-3 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            ref={searchRef}
            value={q}
            onChange={(e) => {
              setPage(1);
              setQ(e.target.value);
            }}
            placeholder="Search email, reason ( / )"
            className="pl-8"
            aria-label="Search requests"
          />
        </div>
      </div>

      {q && (
        <div className="mb-2 flex items-center gap-2 text-xs">
          <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
            <span className="text-muted-foreground">Search: “{q}”</span>
            <button
              className="hover:text-foreground"
              onClick={() => {
                setQ("");
                setPage(1);
              }}
              aria-label="Clear search"
            >
              ×
            </button>
          </span>
        </div>
      )}

      {loading ? (
        <div className="rounded-md border p-10 text-center text-muted-foreground">Loading…</div>
      ) : requests.length === 0 ? (
        <EmptyState
          icon={<UsersIcon className="mx-auto h-12 w-12" />}
          title="No access requests"
          body="Requests from ‘Get access’ will appear here."
        />
      ) : (
        <>
          {selectedIds.length > 0 && (
            <div className="sticky top-0 z-10 mb-3 flex items-center justify-between rounded-lg border border-white/5 bg-background/70 px-3 py-2 text-sm backdrop-blur">
              <div className="font-medium">{selectedIds.length} selected</div>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  className="hover:scale-[1.01]"
                  onClick={() =>
                    selectedIds.forEach((id) => {
                      const r = requests.find((x) => x.id === id);
                      if (r) approveOne(r);
                    })
                  }
                >
                  Approve
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={async () => {
                    setRejectTarget(null);
                    setRejectOpen(true);
                  }}
                >
                  Reject
                </Button>
              </div>
            </div>
          )}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[40px]">
                  <Checkbox
                    checked={requests.every((r) => selected[r.id])}
                    onCheckedChange={(v) => {
                      const next: Record<string, boolean> = { ...selected };
                      requests.forEach((r) => (next[r.id] = Boolean(v)));
                      setSelected(next);
                    }}
                  />
                </TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Submitted</TableHead>
                <TableHead>Reason</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {requests.map((r) => (
                <TableRow
                  key={r.id}
                  className={cn(
                    "odd:bg-foreground/[0.015] hover:bg-foreground/[0.03]",
                    density === "compact" ? "h-11" : "h-14"
                  )}
                >
                  <TableCell className={cn("w-[40px]", density === "compact" && "py-1")}>
                    <Checkbox
                      checked={!!selected[r.id]}
                      onCheckedChange={(v) =>
                        setSelected((prev) => ({ ...prev, [r.id]: Boolean(v) }))
                      }
                    />
                  </TableCell>
                  <TableCell className={cn(density === "compact" && "py-1")}>{r.email}</TableCell>
                  <TableCell className={cn(density === "compact" && "py-1")}>
                    {r.created_at
                      ? formatDistanceToNow(new Date(r.created_at), { addSuffix: true })
                      : "—"}
                  </TableCell>
                  <TableCell
                    className={cn("max-w-[280px] truncate", density === "compact" && "py-1")}
                    title={r.reason || ""}
                  >
                    {r.reason || ""}
                  </TableCell>
                  <TableCell className={cn("text-right", density === "compact" && "py-1")}>
                    <div className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        className="hover:scale-[1.01]"
                        onClick={() => approveOne(r)}
                      >
                        Approve
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setRejectTarget(r);
                          setRejectReason("");
                          setNotify(true);
                          setRejectOpen(true);
                        }}
                      >
                        Reject
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {total !== null && total > pageSize && (
            <div className="mt-3 flex items-center justify-between text-sm">
              <div className="text-muted-foreground">
                Page {page} of {Math.ceil(total / pageSize)}
              </div>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  Previous
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setPage((p) => (total ? (p * pageSize < total ? p + 1 : p) : p))}
                  disabled={total ? page * pageSize >= total : true}
                >
                  Next
                </Button>
              </div>
            </div>
          )}

          {/* Reject confirm modal */}
          <AlertDialog open={rejectOpen}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Reject access request?</AlertDialogTitle>
                <AlertDialogDescription>
                  Optionally include a reason. “Notify requester” is on by default.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <div className="space-y-3">
                <div>
                  <label className="block text-sm mb-1">Reason (optional)</label>
                  <textarea
                    className="w-full rounded-md border bg-background p-2 text-sm"
                    rows={3}
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                  />
                </div>
                <label className="flex items-center gap-2 text-sm">
                  <Checkbox checked={notify} onCheckedChange={(v) => setNotify(Boolean(v))} />
                  Notify requester
                </label>
              </div>
              <AlertDialogFooter>
                <AlertDialogCancel onClick={() => setRejectOpen(false)}>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={async () => {
                    try {
                      if (rejectTarget) {
                        await rejectOne(rejectTarget, rejectReason);
                      } else {
                        const items = requests.filter((r) => selectedIds.includes(r.id));
                        for (const r of items) {
                          await rejectOne(r, rejectReason);
                        }
                      }
                    } finally {
                      setRejectOpen(false);
                    }
                  }}
                >
                  Reject
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </>
      )}
    </div>
  );
}
