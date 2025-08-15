import { useEffect, useMemo, useRef, useState } from "react";
import { Helmet } from "react-helmet-async";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

import { Checkbox } from "@/components/ui/checkbox";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from "@/components/ui/tooltip";
import { useToast } from "@/hooks/use-toast";
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { format, formatDistanceToNow } from "date-fns";
import { Download, Loader2, MoreVertical, Search, Trash2, Upload, Users, X, LogOut, Eye, Power, CheckCircle2, Slash, Mail, Clock, Rows3, List, ChevronsLeft, ChevronsRight, Copy, Laptop, Key, Pencil } from "lucide-react";
import { withLatency } from "@/mocks/latency";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";
import { useAppStore } from "@/store/app-store";

// Types (mocked)
type Credential = {
  id: string;
  label: string;
  addedAt: string;
  lastUsedAt?: string;
  platform: "platform" | "security_key";
  aaguid?: string;
};

export type User = {
  id: string;
  name: string;
  email: string;
  role: "admin" | "member";
  status: "active" | "disabled" | "pending_invite" | "pending_approval";
  createdAt: string; // ISO
  lastActivityAt?: string; // ISO | undefined for never
  sessions: number;
  credentials: Credential[];
  invites?: {
    link: string;
    expiresAt: string;
    lastSentAt: string;
  } | null;
  requests?: Array<{
    submittedAt: string;
    reason?: string;
    status: "pending" | "approved" | "rejected";
    decidedAt?: string;
    decidedBy?: string;
    note?: string;
  }>;
};

// Helpers
const randomFrom = <T,>(arr: T[]) => arr[Math.floor(Math.random() * arr.length)];
const makeId = (prefix: string) => `${prefix}_${Math.random().toString(36).slice(2, 9)}`;
const firstLast = (name: string) => {
  const parts = name.split(" ");
  const first = parts[0]?.[0] ?? "?";
  const last = parts[1]?.[0] ?? parts[0]?.[1] ?? "";
  return (first + last).toUpperCase();
};

function seedUsers(count = 96): User[] {
  const firstNames = ["Alice", "Bob", "Carol", "David", "Eve", "Frank", "Grace", "Heidi", "Ivan", "Judy", "Mallory", "Nia", "Olivia", "Peggy", "Rupert", "Sybil", "Trent", "Victor", "Walter", "Yara", "Zoe"];
  const lastNames = ["Anderson", "Brown", "Clark", "Davis", "Evans", "Foster", "Garcia", "Harris", "Iverson", "Johnson", "Klein", "Lopez", "Miller", "Nguyen", "Olsen", "Patel", "Quinn", "Roberts", "Smith", "Turner", "Ulrich", "Vega", "Williams", "Xu", "Young", "Zimmerman"];
  const roles: Array<User["role"]> = ["admin", "member"];
  const statuses: Array<User["status"]> = ["active", "disabled", "pending_invite", "pending_approval"];
  const users: User[] = [];
  const now = Date.now();

  for (let i = 0; i < count; i++) {
    const fname = randomFrom(firstNames);
    const lname = randomFrom(lastNames);
    const name = `${fname} ${lname}`;
    const email = `${fname}.${lname}${i % 7 === 0 ? ".test" : ""}@example.com`.toLowerCase();
    const role = Math.random() < 0.18 ? "admin" : "member";
    const status = randomFrom(statuses);
    const createdAt = new Date(now - Math.floor(Math.random() * 1000 * 60 * 60 * 24 * 365)).toISOString();
    const lastActivityAt = Math.random() < 0.12 ? undefined : new Date(now - Math.floor(Math.random() * 1000 * 60 * 60 * 24 * 120)).toISOString();
    const sessions = status === "disabled" ? 0 : Math.floor(Math.random() * 4);

    const credsCount = Math.random() < 0.6 ? 1 : Math.random() < 0.8 ? 2 : 0;
    const credentials: Credential[] = Array.from({ length: credsCount }).map(() => ({
      id: makeId("cred"),
      label: Math.random() < 0.5 ? "MacBook Pro" : "YubiKey 5C",
      addedAt: new Date(now - Math.floor(Math.random() * 1000 * 60 * 60 * 24 * 400)).toISOString(),
      lastUsedAt: new Date(now - Math.floor(Math.random() * 1000 * 60 * 60 * 30)).toISOString(),
      platform: Math.random() < 0.6 ? "platform" : "security_key",
      aaguid: Math.random() < 0.5 ? makeId("aag") : undefined,
    }));

    const invites = status === "pending_invite" ? {
      link: `https://app.example.com/invite/${makeId("inv")}`,
      expiresAt: new Date(now + 1000 * 60 * 60 * 24 * 7).toISOString(),
      lastSentAt: new Date(now - 1000 * 60 * 60 * 24).toISOString(),
    } : null;

    const requests = status === "pending_approval" ? [{
      submittedAt: new Date(now - 1000 * 60 * 60 * (Math.floor(Math.random() * 96) + 1)).toISOString(),
      reason: Math.random() < 0.8 ? "Need access to run deployment workflows" : "",
      status: "pending" as const,
    }] : [];

    users.push({ id: makeId("usr"), name, email, role, status, createdAt, lastActivityAt, sessions, credentials, invites, requests });
  }
  return users.sort((a, b) => a.name.localeCompare(b.name));
}

// Status and Role chips
function StatusBadge({ status }: { status: User["status"] }) {
  const map = {
    active: { label: "Active", tooltip: "Active", Icon: CheckCircle2, tone: "success" as const },
    disabled: { label: "Disabled", tooltip: "Disabled account", Icon: Slash, tone: "muted" as const },
    pending_invite: { label: "Pending", tooltip: "Invite sent, awaiting acceptance", Icon: Mail, tone: "primary" as const },
    pending_approval: { label: "Pending", tooltip: "Awaiting admin approval", Icon: Clock, tone: "info" as const },
  } as const;
  const { label, tooltip, Icon, tone } = map[status];
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          aria-label={`Status: ${tooltip}`}
          className={cn(
            "inline-flex items-center justify-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 min-w-[96px]",
            tone === "success" && "bg-success/20 text-success/80 ring-success/30",
            tone === "muted" && "bg-muted text-muted-foreground ring-border/30",
            tone === "primary" && "bg-primary/20 text-primary/80 ring-primary/30",
            tone === "info" && "bg-info/30 text-info/95 ring-info/50"
          )}
        >
          <Icon className="h-3.5 w-3.5" />
          <span>{label}</span>
        </span>
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}

function RoleBadge({ role }: { role: User["role"] }) {
  return (
    <span aria-label={`Role: ${role}`} className="rounded-full border border-muted-foreground/60 px-2 py-0.5 text-xs text-muted-foreground">
      {role === "admin" ? "Admin" : "Member"}
    </span>
  );
}

function HeaderUnderline() {
  return <div className="mt-2 h-0.5 w-32 rounded-full bg-gradient-to-r from-primary via-primary/60 to-transparent" />;
}

export default function UsersPage() {
  const [loading, setLoading] = useState(true);
  const [users, setUsers] = useState<User[]>([]);
  const [openUserId, setOpenUserId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [activityFilter, setActivityFilter] = useState<string>("all");
  const [density, setDensity] = useState<"comfortable" | "compact">("comfortable");
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [sort, setSort] = useState<{ key: "name" | "lastActivityAt" | "createdAt"; dir: "asc" | "desc" }>({ key: "name", dir: "asc" });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const searchRef = useRef<HTMLInputElement>(null);
  const requestsSearchRef = useRef<HTMLInputElement>(null);
  const [activeTab, setActiveTab] = useState<"directory" | "requests">("directory");
  const { toast } = useToast();
  const { state } = useAppStore();
  
  // Seed data
  useEffect(() => {
    (async () => {
      await withLatency(350, 700);
      setUsers(seedUsers(120));
      setLoading(false);
    })();
  }, []);

  // Persist density preference
  useEffect(() => {
    const saved = localStorage.getItem("users:density");
    if (saved === "comfortable" || saved === "compact") setDensity(saved as any);
  }, []);
  useEffect(() => {
    localStorage.setItem("users:density", density);
  }, [density]);

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "/") {
        e.preventDefault();
        (activeTab === "directory" ? searchRef : requestsSearchRef).current?.focus();
      }
      if (e.shiftKey && (e.key.toLowerCase() === "a")) {
        e.preventDefault();
        setInviteOpen(true);
      }
      if (e.shiftKey && (e.key.toLowerCase() === "d")) {
        e.preventDefault();
        setDensity((prev) => (prev === "compact" ? "comfortable" : "compact"));
      }
      if (e.key === "Escape" && openUserId) {
        setOpenUserId(null);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [openUserId, activeTab]);

  // Derived rows
  const filtered = useMemo(() => {
    const s = search.trim().toLowerCase();
    let list = users.filter(u =>
      !s || u.name.toLowerCase().includes(s) || u.email.toLowerCase().includes(s) || u.id.toLowerCase().includes(s)
    );

    if (statusFilter !== "all") list = list.filter(u => u.status === statusFilter);
    if (roleFilter !== "all") list = list.filter(u => u.role === roleFilter);
    if (activityFilter !== "all") {
      const now = Date.now();
      const ranges: Record<string, number> = {
        "24h": 24,
        "7d": 24 * 7,
        "30d": 24 * 30,
        "90d": 24 * 90,
      };
      if (activityFilter === "never") {
        list = list.filter(u => !u.lastActivityAt);
      } else if (ranges[activityFilter]) {
        const hours = ranges[activityFilter];
        const cutoff = now - hours * 60 * 60 * 1000;
        list = list.filter(u => (u.lastActivityAt ? new Date(u.lastActivityAt).getTime() >= cutoff : false));
      }
    }

    const sorted = [...list].sort((a, b) => {
      let res = 0;
      if (sort.key === "name") res = a.name.localeCompare(b.name);
      if (sort.key === "lastActivityAt") {
        const at = a.lastActivityAt ? new Date(a.lastActivityAt).getTime() : -Infinity;
        const bt = b.lastActivityAt ? new Date(b.lastActivityAt).getTime() : -Infinity;
        res = at - bt;
      }
      if (sort.key === "createdAt") res = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
      return sort.dir === "asc" ? res : -res;
    });

    return sorted;
  }, [users, search, statusFilter, roleFilter, activityFilter, sort]);

  const total = filtered.length;
  const totalAll = users.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const startIdx = (page - 1) * pageSize;
  const endIdx = Math.min(startIdx + pageSize, total);
  const visible = filtered.slice(startIdx, endIdx);
  const hiddenCount = Math.max(0, users.length - total);
  const showFilterBanner = users.length > 0 && total > 0 && hiddenCount / users.length > 0.8;

  useEffect(() => {
    if (page > pageCount) setPage(1);
  }, [pageCount, page]);

  // Selection helpers
  const selectedIds = Object.keys(selected).filter((id) => selected[id]);
  const allVisibleSelected = visible.length > 0 && visible.every((u) => selected[u.id]);

  const toggleAllVisible = (checked: boolean) => {
    const next = { ...selected };
    visible.forEach((u) => (next[u.id] = checked));
    setSelected(next);
  };

  // Actions (mocked)
  const bulkDisable = () => {
    setUsers((prev) => prev.map((u) => (selected[u.id] ? { ...u, status: "disabled", sessions: 0 } : u)));
    toast({ title: `${selectedIds.length} users disabled.` });
    setSelected({});
  };
  const bulkEnable = () => {
    setUsers((prev) => prev.map((u) => (selected[u.id] ? { ...u, status: "active" } : u)));
    toast({ title: `${selectedIds.length} users re-enabled.` });
    setSelected({});
  };
  const bulkLogout = () => {
    setUsers((prev) => prev.map((u) => (selected[u.id] ? { ...u, sessions: 0 } : u)));
    toast({ title: `Forced log out for ${selectedIds.length} users.` });
    setSelected({});
  };
  const bulkDelete = () => {
    setUsers((prev) => prev.filter((u) => !selected[u.id]));
    toast({ title: `${selectedIds.length} users deleted.` });
    setSelected({});
  };
  const changeRoleBulk = (role: User["role"]) => {
    setUsers((prev) => prev.map((u) => (selected[u.id] ? { ...u, role } : u)));
    toast({ title: `Changed role to ${role} for ${selectedIds.length}.` });
    setSelected({});
  };

  // CSV Export
  const exportCsv = () => {
    const rows = filtered.map((u) => ({
      id: u.id,
      name: u.name,
      email: u.email,
      role: u.role,
      status: u.status,
      createdAt: u.createdAt,
      lastActivityAt: u.lastActivityAt ?? "",
      sessions: u.sessions,
    }));
    const header = Object.keys(rows[0] ?? { id: "", name: "", email: "", role: "", status: "", createdAt: "", lastActivityAt: "", sessions: 0 }).join(",");
    const body = rows.map((r) => Object.values(r).map((v) => `${String(v).replace(/"/g, '""')}`).join(",")).join("\n");
    const csv = `${header}\n${body}`;
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `users_export_${format(new Date(), "yyyyMMdd_HHmmss")}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  // Invite flow (mock)
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<User["role"]>("member");
  const [inviteDays, setInviteDays] = useState("7");

  const submitInvite = async () => {
    if (!inviteEmail) return;
    const now = Date.now();
    const newUser: User = {
      id: makeId("usr"),
      name: inviteEmail.split("@")[0].replace(/\./g, " ").replace(/\b\w/g, (c) => c.toUpperCase()) || "Pending User",
      email: inviteEmail,
      role: inviteRole,
      status: "pending_invite",
      createdAt: new Date(now).toISOString(),
      lastActivityAt: undefined,
      sessions: 0,
      credentials: [],
      invites: {
        link: `https://app.example.com/invite/${makeId("inv")}`,
        expiresAt: new Date(now + parseInt(inviteDays) * 24 * 60 * 60 * 1000).toISOString(),
        lastSentAt: new Date(now).toISOString(),
      },
      requests: [],
    };
    setUsers((prev) => [newUser, ...prev]);
    setInviteOpen(false);
    setInviteEmail("");
    toast({ title: `Invite sent to ${newUser.email}. Link copied to clipboard.` });
    try {
      await navigator.clipboard.writeText(newUser.invites!.link);
    } catch {}
  };

  // User sheet data
  const openUser = users.find((u) => u.id === openUserId) || null;

  return (
    <div className="p-4 md:p-6">
      <Helmet>
        <title>Users | Admin Console</title>
        <meta name="description" content="Manage and audit users: search, filter, view details, and take actions." />
        <link rel="canonical" href="/users" />
      </Helmet>

      <header className="mb-4 md:mb-6">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl md:text-3xl font-semibold tracking-tight">Users <span className="ml-2 align-middle text-xs font-normal text-muted-foreground">{totalAll.toLocaleString()}</span></h1>
            <HeaderUnderline />
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" onClick={exportCsv} aria-label="Export CSV">
              <Download className="h-4 w-4" />
              <span className="hidden sm:inline">Export CSV</span>
            </Button>
            <Button variant="hero" onClick={() => setInviteOpen(true)} aria-label="Invite user">
              <Upload className="h-4 w-4" />
              <span className="hidden sm:inline">Invite user</span>
            </Button>
          </div>
        </div>
      </header>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as any)}>
          <div className="flex items-center gap-3 flex-wrap">
            <TabsList>
              <TabsTrigger value="directory">Directory</TabsTrigger>
              <TabsTrigger value="requests">Requests</TabsTrigger>
            </TabsList>
            <div className="hidden sm:block h-6 w-px bg-border/60" aria-hidden />
            <ToggleGroup
              type="single"
              value={density}
              onValueChange={(v) => v && setDensity(v as any)}
              size="sm"
              className="inline-flex rounded-lg border border-input bg-muted/30 p-0.5"
            >
              <TooltipProvider>
                <div className="flex items-center gap-0">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <ToggleGroupItem
                        value="comfortable"
                        aria-label="Comfortable view"
                        variant="default"
                        className={cn(
                          "h-8 px-3 rounded-md text-sm inline-flex items-center gap-1.5",
                          "text-muted-foreground border border-transparent",
                          "hover:bg-accent hover:text-foreground",
                          "data-[state=on]:bg-background data-[state=on]:text-foreground",
                          "data-[state=on]:border-input data-[state=on]:shadow-sm"
                        )}
                      >
                        <Rows3 className="h-4 w-4" />
                        <span className="ml-1 hidden sm:inline">Comfortable</span>
                      </ToggleGroupItem>
                    </TooltipTrigger>
                    <TooltipContent>Comfortable view</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <ToggleGroupItem
                        value="compact"
                        aria-label="Compact view"
                        variant="default"
                        className={cn(
                          "h-8 px-3 rounded-md text-sm inline-flex items-center gap-1.5",
                          "text-muted-foreground border border-transparent",
                          "hover:bg-accent hover:text-foreground",
                          "data-[state=on]:bg-background data-[state=on]:text-foreground",
                          "data-[state=on]:border-input data-[state=on]:shadow-sm"
                        )}
                      >
                        <List className="h-4 w-4" />
                        <span className="ml-1 hidden sm:inline">Compact</span>
                      </ToggleGroupItem>
                    </TooltipTrigger>
                    <TooltipContent>Compact view</TooltipContent>
                  </Tooltip>
                </div>
              </TooltipProvider>
            </ToggleGroup>
          </div>
        <TabsContent value="directory" className="mt-4">
          {/* Controls */}
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="flex flex-1 items-center gap-2">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input ref={searchRef} value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search name, email, ID ( / )" className="pl-8" aria-label="Search users" />
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-[160px]"><SelectValue placeholder="Status" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Status: All</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="disabled">Disabled</SelectItem>
                  <SelectItem value="pending_invite">Pending invite</SelectItem>
                  <SelectItem value="pending_approval">Pending approval</SelectItem>
                </SelectContent>
              </Select>
              <Select value={roleFilter} onValueChange={setRoleFilter}>
                <SelectTrigger className="w-[140px]"><SelectValue placeholder="Role" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Role: All</SelectItem>
                  <SelectItem value="admin">Admin</SelectItem>
                  <SelectItem value="member">Member</SelectItem>
                </SelectContent>
              </Select>
              <Select value={activityFilter} onValueChange={setActivityFilter}>
                <SelectTrigger className="w-[160px]"><SelectValue placeholder="Last activity" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Last activity: All</SelectItem>
                  <SelectItem value="24h">24h</SelectItem>
                  <SelectItem value="7d">7d</SelectItem>
                  <SelectItem value="30d">30d</SelectItem>
                  <SelectItem value="90d">90d</SelectItem>
                  <SelectItem value="never">Never</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Active filter chips */}
          {(statusFilter !== "all" || roleFilter !== "all" || activityFilter !== "all") && (
            <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
              {statusFilter !== "all" && (
                <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
                  <span className="text-muted-foreground">Status: {statusFilter.replace("_", " ")}</span>
                  <button className="hover:text-foreground" onClick={() => setStatusFilter("all")} aria-label="Clear status">×</button>
                </span>
              )}
              {roleFilter !== "all" && (
                <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
                  <span className="text-muted-foreground">Role: {roleFilter}</span>
                  <button className="hover:text-foreground" onClick={() => setRoleFilter("all")} aria-label="Clear role">×</button>
                </span>
              )}
              {activityFilter !== "all" && (
                <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
                  <span className="text-muted-foreground">Last activity: {activityFilter}</span>
                  <button className="hover:text-foreground" onClick={() => setActivityFilter("all")} aria-label="Clear last activity">×</button>
                </span>
              )}
              <button className="ml-auto text-muted-foreground hover:text-foreground" onClick={() => { setStatusFilter("all"); setRoleFilter("all"); setActivityFilter("all"); }}>Clear all</button>
            </div>
          )}

          {/* Selection bar */}
          {selectedIds.length > 0 && (
            <div className="mt-3 flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2 text-sm">
              <div>
                <span className="font-medium">{selectedIds.length} selected</span>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Button size="sm" variant="secondary" onClick={bulkDisable}>Disable</Button>
                <Button size="sm" variant="secondary" onClick={bulkEnable}>Re-enable</Button>
                <Button size="sm" variant="secondary" onClick={bulkLogout}><LogOut className="h-4 w-4" /> Force log out</Button>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button size="sm" variant="outline">Change role</Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => changeRoleBulk("admin")}>Admin</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => changeRoleBulk("member")}>Member</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                <Confirm destructive label="Delete" onConfirm={bulkDelete} description="This removes selected users and their credentials (mock)." />
              </div>
            </div>
          )}

          {/* Table */}
          <div className="mt-3 rounded-md border">
            {showFilterBanner && (
              <div className="m-3 rounded-md bg-foreground/[0.03] px-3 py-2 text-sm text-muted-foreground">
                Filters hide {hiddenCount.toLocaleString()} users.
              </div>
            )}
            {loading ? (
              <div className="flex items-center justify-center p-10 text-muted-foreground">
                <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading users…
              </div>
            ) : totalAll === 0 ? (
              <EmptyState
                icon={<Users className="mx-auto h-12 w-12" />}
                title="No users yet"
                body="Invite your first teammate to Catalyst Forge."
                primaryCta={{ label: "Invite user", onClick: () => setInviteOpen(true) }}
              />
            ) : total === 0 ? (
              <EmptyState
                icon={<Users className="mx-auto h-12 w-12" />}
                title="Nothing matches your filters"
                primaryCta={{ label: "Clear filters", onClick: () => { setSearch(""); setStatusFilter("all"); setRoleFilter("all"); setActivityFilter("all"); } }}
                secondaryCta={{ label: "Invite user", onClick: () => setInviteOpen(true) }}
              />
            ) : (
              <Table className="">
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[40px]">
                      <Checkbox checked={allVisibleSelected} onCheckedChange={(v) => toggleAllVisible(Boolean(v))} aria-label="Select all visible" />
                    </TableHead>
                    <TableHead className="cursor-pointer select-none" onClick={() => setSort((s) => ({ key: "name", dir: s.key === "name" && s.dir === "asc" ? "desc" : "asc" }))}>
                      User {sort.key === "name" && (sort.dir === "asc" ? "↑" : "↓")}
                    </TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead className="cursor-pointer select-none" onClick={() => setSort((s) => ({ key: "lastActivityAt", dir: s.key === "lastActivityAt" && s.dir === "asc" ? "desc" : "asc" }))}>
                      Last activity {sort.key === "lastActivityAt" && (sort.dir === "asc" ? "↑" : "↓")}
                    </TableHead>
                    <TableHead>Sessions</TableHead>
                    <TableHead className="cursor-pointer select-none" onClick={() => setSort((s) => ({ key: "createdAt", dir: s.key === "createdAt" && s.dir === "asc" ? "desc" : "asc" }))}>
                      Created {sort.key === "createdAt" && (sort.dir === "asc" ? "↑" : "↓")}
                    </TableHead>
                    <TableHead className="w-[120px] text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visible.map((u, idx) => (
                    <TableRow
                      key={u.id}
                      className={cn(
                        "cursor-pointer group",
                        density === "compact" ? "h-11" : "h-14",
                        "odd:bg-foreground/[0.015] even:bg-transparent hover:bg-foreground/[0.03]"
                      )}
                      onClick={(e) => {
                        const tag = (e.target as HTMLElement).closest("button,input,svg,span,[role='menuitem']");
                        if (tag) return;
                        setOpenUserId(u.id);
                      }}
                    >
                      <TableCell className={cn("w-[40px]", density === "compact" && "py-1")} onClick={(e) => e.stopPropagation()}>
                        <Checkbox
                          checked={!!selected[u.id]}
                          onCheckedChange={(v) => setSelected((prev) => ({ ...prev, [u.id]: Boolean(v) }))}
                          aria-label={`Select ${u.name}`}
                        />
                      </TableCell>
                      <TableCell className={cn(density === "compact" && "py-1")}> 
                        <div className="flex items-center gap-3">
                          <Avatar className="h-9 w-9">
                            <AvatarFallback>{firstLast(u.name)}</AvatarFallback>
                          </Avatar>
                          <div>
                            <div className={cn("font-medium", density === "compact" ? "text-sm" : "")}>{(u.name && u.name.trim() && u.name.trim().toLowerCase() !== u.email.toLowerCase()) ? u.name : u.email}</div>
                            {(u.name && u.name.trim() && u.name.trim().toLowerCase() !== u.email.toLowerCase()) && (
                              <div className="text-xs text-muted-foreground">{u.email}</div>
                            )}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className={cn(density === "compact" && "py-1")}> <StatusBadge status={u.status} /> </TableCell>
                      <TableCell className={cn(density === "compact" && "py-1")}> <RoleBadge role={u.role} /> </TableCell>
                      <TableCell className={cn(density === "compact" && "py-1")} title={u.lastActivityAt ? format(new Date(u.lastActivityAt), "PPpp") : "Never"}>
                        {u.lastActivityAt ? `${formatDistanceToNow(new Date(u.lastActivityAt), { addSuffix: true })}` : "Never"}
                      </TableCell>
                      <TableCell className={cn(density === "compact" && "py-1")}>{u.sessions}</TableCell>
                      <TableCell className={cn(density === "compact" && "py-1")} title={format(new Date(u.createdAt), "PPpp")}>{format(new Date(u.createdAt), "PP")}</TableCell>
                      <TableCell className={cn("text-right", density === "compact" && "py-1")} onClick={(e) => e.stopPropagation()}>
                        <TooltipProvider>
                          <div className="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="ghost" size="icon" aria-label="View" onClick={() => setOpenUserId(u.id)}>
                                  <Eye className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>View</TooltipContent>
                            </Tooltip>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="ghost" size="icon" aria-label={u.status === "disabled" ? "Re-enable" : "Disable"} onClick={() => setUsers((prev) => prev.map((x) => x.id === u.id ? { ...x, status: x.status === "disabled" ? "active" : "disabled", sessions: 0 } : x))}>
                                  <Power className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>{u.status === "disabled" ? "Re-enable" : "Disable"}</TooltipContent>
                            </Tooltip>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="ghost" size="icon" aria-label="Force log out" onClick={() => setUsers((prev) => prev.map((x) => x.id === u.id ? { ...x, sessions: 0 } : x))}>
                                  <LogOut className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Force log out</TooltipContent>
                            </Tooltip>
                            <DropdownMenu>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <DropdownMenuTrigger asChild>
                                    <Button variant="ghost" size="icon" aria-label="More">
                                      <MoreVertical className="h-4 w-4" />
                                    </Button>
                                  </DropdownMenuTrigger>
                                </TooltipTrigger>
                                <TooltipContent>More</TooltipContent>
                              </Tooltip>
                              <DropdownMenuContent align="end">
                                {u.status === "pending_invite" && (
                                  <DropdownMenuItem onClick={() => toast({ title: `Invite resent to ${u.email}.` })}>Resend invite</DropdownMenuItem>
                                )}
                                <DropdownMenuItem onClick={() => setUsers((prev) => prev.map((x) => x.id === u.id ? { ...x, role: x.role === "admin" ? "member" : "admin" } : x))}>Toggle role</DropdownMenuItem>
                                <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={() => setUsers((prev) => prev.filter((x) => x.id !== u.id))}><Trash2 className="h-4 w-4 mr-2" /> Delete</DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </div>
                        </TooltipProvider>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>

          {/* Pagination */}
          {!loading && total > 0 && (
            <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div className="text-sm text-muted-foreground">
                {total === 0 ? "0" : `${startIdx + 1}–${endIdx}`} of {total.toLocaleString()}
              </div>
              <div className="flex items-center gap-3">
                <Select value={String(pageSize)} onValueChange={(v) => { setPageSize(parseInt(v)); setPage(1); }}>
                  <SelectTrigger className="w-[120px] shrink-0 whitespace-nowrap"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="25">25 / page</SelectItem>
                    <SelectItem value="50">50 / page</SelectItem>
                    <SelectItem value="100">100 / page</SelectItem>
                  </SelectContent>
                </Select>
                <Pagination>
                  <PaginationContent>
                    <PaginationItem>
                      <PaginationLink
                        href="#"
                        className="gap-1 pl-2.5"
                        onClick={(e) => { e.preventDefault(); if (page > 1) setPage(1); }}
                        aria-label="Go to first page"
                        aria-disabled={page === 1}
                        size="default"
                      >
                        <ChevronsLeft className="h-4 w-4" />
                        <span>First</span>
                      </PaginationLink>
                    </PaginationItem>
                    <PaginationItem>
                      <PaginationPrevious href="#" onClick={(e) => { e.preventDefault(); setPage((p) => Math.max(1, p - 1)); }} aria-disabled={page === 1} />
                    </PaginationItem>
                    <PaginationItem>
                      <PaginationLink href="#" isActive>{page}</PaginationLink>
                    </PaginationItem>
                    <PaginationItem>
                      <PaginationNext href="#" onClick={(e) => { e.preventDefault(); setPage((p) => Math.min(pageCount, p + 1)); }} aria-disabled={page === pageCount} />
                    </PaginationItem>
                    <PaginationItem>
                      <PaginationLink
                        href="#"
                        className="gap-1 pr-2.5"
                        onClick={(e) => { e.preventDefault(); if (page < pageCount) setPage(pageCount); }}
                        aria-label="Go to last page"
                        aria-disabled={page === pageCount}
                        size="default"
                      >
                        <span>Last</span>
                        <ChevronsRight className="h-4 w-4" />
                      </PaginationLink>
                    </PaginationItem>
                  </PaginationContent>
                </Pagination>
              </div>
            </div>
          )}
        </TabsContent>

        {/* Requests Tab (simplified mocked) */}
        <TabsContent value="requests" className="mt-4">
          <RequestsTab users={users} setUsers={setUsers} density={density} setDensity={setDensity} searchRef={requestsSearchRef} />
        </TabsContent>
      </Tabs>

      {/* Invite dialog */}
      <AlertDialog open={inviteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Invite user</AlertDialogTitle>
            <AlertDialogDescription>Send an invitation link with an optional expiration.</AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-3">
            <div>
              <label className="block text-sm mb-1">Email</label>
              <Input type="email" value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} placeholder="user@company.com" />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-sm mb-1">Role</label>
                <Select value={inviteRole} onValueChange={(v) => setInviteRole(v as any)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="member">Member</SelectItem>
                    <SelectItem value="admin">Admin</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label className="block text-sm mb-1">Expiration</label>
                <Select value={inviteDays} onValueChange={setInviteDays}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="7">7 days</SelectItem>
                    <SelectItem value="14">14 days</SelectItem>
                    <SelectItem value="30">30 days</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setInviteOpen(false)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={submitInvite}
              disabled={!inviteEmail || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(inviteEmail)}
              className="disabled:opacity-80 disabled:bg-primary/35 disabled:text-foreground/80 disabled:cursor-not-allowed hover:scale-[1.01] shadow-[0_8px_24px_-8px_hsl(var(--primary)/0.25)]"
            >
              Send invite
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* User slide-over */}
      <Sheet open={!!openUser} onOpenChange={(o) => setOpenUserId(o ? openUserId : null)}>
        <SheetContent side="right" className="w-full sm:max-w-[520px] md:max-w-[560px] motion-safe:animate-in fade-in slide-in-from-right-2">
          {openUser && (
            <div className="flex h-full flex-col">
              <SheetHeader className="sticky top-0 z-10 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                <SheetTitle>
                  <div className="rounded-lg border border-white/5 bg-foreground/[0.02] shadow-inner">
                    <div className={cn("grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3", density === "compact" ? "px-4 pt-4 pb-3" : "px-5 pt-5 pb-4")}>
                      {/* Identity */}
                      <div className="min-w-0 flex items-center gap-3">
                        <Avatar className="h-9 w-9">
                          <AvatarFallback>{firstLast(openUser.name)}</AvatarFallback>
                        </Avatar>
                        <div className="min-w-0">
                          <div className="truncate font-semibold leading-tight">
                            {(openUser.name && openUser.name.trim().toLowerCase() !== openUser.email.toLowerCase())
                              ? openUser.name
                              : openUser.email}
                          </div>
                          {(openUser.name && openUser.name.trim().toLowerCase() !== openUser.email.toLowerCase()) && (
                            <div className="truncate text-xs text-muted-foreground">{openUser.email}</div>
                          )}
                          <div className="mt-1 flex flex-wrap items-center gap-2">
                            <RoleBadge role={openUser.role} />
                            <StatusBadge status={openUser.status} />
                          </div>
                        </div>
                      </div>

                      {/* Overflow only */}
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button size="icon" variant="ghost" className="h-8 w-8 shrink-0">
                            <MoreVertical className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {openUser.status === "disabled" ? (
                            <DropdownMenuItem onClick={() => setUsers((prev) => prev.map((u) => u.id === openUser.id ? { ...u, status: "active" } : u))}>
                              Enable
                            </DropdownMenuItem>
                          ) : (
                            <DropdownMenuItem onClick={() => setUsers((prev) => prev.map((u) => u.id === openUser.id ? { ...u, status: "disabled", sessions: 0 } : u))}>
                              Disable
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuItem onClick={() => setUsers((prev) => prev.map((u) => u.id === openUser.id ? { ...u, sessions: 0 } : u))}>
                            Log out
                          </DropdownMenuItem>
                          {openUser.status === "pending_invite" && (
                            <DropdownMenuItem onClick={() => toast({ title: `Invite resent to ${openUser.email}.` })}>
                              Resend invite
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuItem onClick={() => setUsers((prev) => prev.map((u) => u.id === openUser.id ? { ...u, role: u.role === "admin" ? "member" : "admin" } : u))}>
                            Toggle role
                          </DropdownMenuItem>
                          <DropdownMenuItem className="text-destructive" onClick={() => setUsers((prev) => prev.filter((u) => u.id !== openUser.id))}>
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                </SheetTitle>
              </SheetHeader>

              <div className="flex-1 overflow-y-auto">
                {/* Overview */}
                <section className={cn("border-b border-white/[0.06]", density === "compact" ? "px-4 py-3" : "px-5 py-4")}> 
                  <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Overview</h3>
                  <HeaderUnderline />
                  <div className="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
                    <div>
                      <div className="text-muted-foreground">User ID</div>
                      <div className="flex items-center gap-2 font-mono text-xs break-all">
                        <span className="truncate">{openUser.id}</span>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button size="icon" variant="ghost" aria-label="Copy user ID" onClick={async () => { try { await navigator.clipboard.writeText(openUser.id); toast({ title: "Copied user ID" }); } catch {} }}>
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
                      <div><RoleBadge role={openUser.role} /></div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">Created</div>
                      <div>{format(new Date(openUser.createdAt), "PPpp")}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">Active sessions</div>
                      <div>{openUser.sessions}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">Last login</div>
                      <div>{openUser.lastActivityAt ? format(new Date(openUser.lastActivityAt), "PPpp") : "Never"}</div>
                    </div>
                  </div>
                </section>

                {/* Credentials */}
                <section className={cn("border-b border-white/[0.06]", density === "compact" ? "px-4 py-3" : "px-5 py-4")}> 
                  <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Credentials</h3>
                  <HeaderUnderline />
                  <div className="mt-3 space-y-2">
                    {openUser.credentials.length === 0 ? (
                      <div className="text-sm text-muted-foreground">No credentials added.</div>
                    ) : (
                      openUser.credentials.map((c) => (
                        <article key={c.id} className="flex items-center justify-between rounded-md border bg-background/[0.6] px-3 py-2">
                          <div className="flex items-center gap-3">
                            {c.platform === "platform" ? <Laptop className="h-4 w-4 text-muted-foreground" /> : <Key className="h-4 w-4 text-muted-foreground" />}
                            <div className="text-sm">
                              <div className="font-medium">{c.label}</div>
                              <div className="text-xs text-muted-foreground">Added {format(new Date(c.addedAt), "PP")} • Last used {c.lastUsedAt ? format(new Date(c.lastUsedAt), "PP") : "—"} • {c.platform === "platform" ? "Platform" : "Security key"}</div>
                            </div>
                          </div>
                          <div className="flex items-center gap-1">
                            <TooltipProvider>
                              <Tooltip><TooltipTrigger asChild>
                                <Button size="icon" variant="ghost" aria-label="Rename" onClick={() => toast({ title: "Rename (mock)", description: "Not implemented in mocks" })}>
                                  <Pencil className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger><TooltipContent>Rename</TooltipContent></Tooltip>
                            </TooltipProvider>
                            <TooltipProvider>
                              <Tooltip><TooltipTrigger asChild>
                                <Button size="icon" variant="ghost" aria-label="Remove" className="hover:text-destructive" onClick={() => setUsers((prev) => prev.map((u) => u.id === openUser.id ? { ...u, credentials: u.credentials.filter((x) => x.id !== c.id) } : u))}>
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger><TooltipContent>Remove</TooltipContent></Tooltip>
                            </TooltipProvider>
                          </div>
                        </article>
                      ))
                    )}
                    <Button size="sm" variant="outline" onClick={() => setUsers((prev) => prev.map((u) => u.id === openUser.id ? { ...u, credentials: [...u.credentials, { id: makeId("cred"), label: "New credential", addedAt: new Date().toISOString(), platform: "platform" }] } : u))}>+ Add credential</Button>
                  </div>
                </section>

                {/* Security */}
                <section className={cn("border-b border-white/[0.06]", density === "compact" ? "px-4 py-3" : "px-5 py-4")}> 
                  <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Security</h3>
                  <HeaderUnderline />
                  <div className="mt-3 flex items-center justify-between">
                    <div className="text-sm text-muted-foreground">Recovery keys: Regenerated {format(new Date(openUser.createdAt), "PP")} (mock)</div>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span>
                            <Button size="sm" disabled={openUser.status === "disabled"} onClick={() => toast({ title: "Recovery keys regenerated (mock)" })}>
                              Regenerate
                            </Button>
                          </span>
                        </TooltipTrigger>
                        {openUser.status === "disabled" && (<TooltipContent>Action not available while user is disabled</TooltipContent>)}
                      </Tooltip>
                    </TooltipProvider>
                  </div>
                </section>


                {/* Audit */}
                <section className={cn(density === "compact" ? "px-4 py-3" : "px-5 py-4")}> 
                  <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Audit</h3>
                  <HeaderUnderline />
                  <div className="mt-3">
                    <div className="overflow-x-auto">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="text-xs text-muted-foreground">
                            <th className="text-left font-normal">Actor</th>
                            <th className="text-left font-normal">Action</th>
                            <th className="text-left font-normal">Target</th>
                            <th className="text-left font-normal">Date</th>
                          </tr>
                        </thead>
                        <tbody>
                          {(state.audit.slice(0, 4)).map((evt) => (
                            <tr key={evt.id} className="border-t border-white/[0.06]">
                              <td className="py-2">{evt.actor}</td>
                              <td className="py-2">{evt.action}</td>
                              <td className="py-2">{evt.resource}</td>
                              <td className="py-2">{format(new Date(evt.timestamp), "PP")}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    <div className="mt-2 text-right">
                      <a href="/audit" className="text-sm story-link">Open in Audit Log</a>
                    </div>
                  </div>
                </section>
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}

function RowActions({ user, onOpen, onChange, onDelete }: { user: User; onOpen: () => void; onChange: (u: User) => void; onDelete: () => void }) {
  const { toast } = useToast();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Row actions">
          <MoreVertical className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onOpen}>View</DropdownMenuItem>
        {user.status === "disabled" ? (
          <DropdownMenuItem onClick={() => onChange({ ...user, status: "active" })}>Re-enable</DropdownMenuItem>
        ) : (
          <DropdownMenuItem onClick={() => onChange({ ...user, status: "disabled", sessions: 0 })}>Disable</DropdownMenuItem>
        )}
        <DropdownMenuItem onClick={() => onChange({ ...user, sessions: 0 })}>Force log out</DropdownMenuItem>
        {user.status === "pending_invite" && (
          <DropdownMenuItem onClick={() => toast({ title: `Invite resent to ${user.email}.` })}>Resend invite</DropdownMenuItem>
        )}
        <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={onDelete}><Trash2 className="h-4 w-4 mr-2" /> Delete</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function Confirm({ label, description, destructive, onConfirm }: { label: string; description?: string; destructive?: boolean; onConfirm: () => void }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button size="sm" variant={destructive ? "destructive" : "secondary"} onClick={() => setOpen(true)}>{label}</Button>
      <AlertDialog open={open}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{label}</AlertDialogTitle>
            {description && <AlertDialogDescription>{description}</AlertDialogDescription>}
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setOpen(false)}>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => { onConfirm(); setOpen(false); }}>Confirm</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

// Requests tab (polished mock)
function RequestsTab({ users, setUsers, density, setDensity, searchRef }: { users: User[]; setUsers: React.Dispatch<React.SetStateAction<User[]>>; density: "comfortable" | "compact"; setDensity: (d: "comfortable" | "compact") => void; searchRef: React.RefObject<HTMLInputElement>; }) {
  const { toast } = useToast();
  const base = users.filter((u) => u.status === "pending_approval");
  const [q, setQ] = useState("");
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectTarget, setRejectTarget] = useState<User | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [notify, setNotify] = useState(true);

  const pending = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return base;
    return base.filter((u) => {
      const reason = u.requests?.[0]?.reason || "";
      return (
        u.name.toLowerCase().includes(s) ||
        u.email.toLowerCase().includes(s) ||
        u.id.toLowerCase().includes(s) ||
        reason.toLowerCase().includes(s)
      );
    });
  }, [base, q]);

  const approveOne = (u: User) => {
    setUsers((prev) => prev.map((x) => x.id === u.id ? { ...x, status: "pending_invite", invites: { link: `https://app.example.com/invite/${makeId("inv")}`, expiresAt: new Date(Date.now() + 7 * 86400e3).toISOString(), lastSentAt: new Date().toISOString() } } : x));
    toast({ title: `Access granted to ${u.email}. Invitation sent.` });
  };
  const rejectOne = (u: User, reason?: string) => {
    setUsers((prev) => prev.map((x) => x.id === u.id ? { ...x, status: "disabled", requests: [{ ...(x.requests?.[0] ?? { submittedAt: new Date().toISOString() }), status: "rejected", decidedAt: new Date().toISOString(), decidedBy: "admin", note: reason || "" }] } : x));
    toast({ title: `Request rejected for ${u.email}.` });
  };

  const selectedIds = Object.keys(selected).filter((id) => selected[id]);

  return (
    <div>
      <div className="mb-3 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input ref={searchRef} value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search name, email, reason ( / )" className="pl-8" aria-label="Search requests" />
        </div>
      </div>

      {q && (
        <div className="mb-2 flex items-center gap-2 text-xs">
          <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
            <span className="text-muted-foreground">Search: “{q}”</span>
            <button className="hover:text-foreground" onClick={() => setQ("")} aria-label="Clear search">×</button>
          </span>
        </div>
      )}

      {base.length === 0 ? (
        <EmptyState
          icon={<Users className="mx-auto h-12 w-12" />}
          title="No access requests"
          body="Requests from ‘Get access’ will appear here."
        />
      ) : (
        <>
          {selectedIds.length > 0 && (
            <div className="sticky top-0 z-10 mb-3 flex items-center justify-between rounded-lg border border-white/5 bg-background/70 px-3 py-2 text-sm backdrop-blur">
              <div className="font-medium">{selectedIds.length} selected</div>
              <div className="flex items-center gap-2">
                <Button size="sm" className="hover:scale-[1.01]" onClick={() => selectedIds.forEach((id) => approveOne(users.find(u => u.id === id)!))}>Approve</Button>
                <Button size="sm" variant="outline" onClick={() => { setRejectTarget(null); setRejectOpen(true); }}>
                  Reject
                </Button>
              </div>
            </div>
          )}
          {pending.length === 0 ? (
            <div className="rounded-md border p-10 text-center text-muted-foreground">
              <div className="opacity-30 mb-1">No matching requests.</div>
              <button className="text-sm story-link" onClick={() => setQ("")}>Clear search</button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[40px]"><Checkbox checked={pending.every((u) => selected[u.id])} onCheckedChange={(v) => { const next: Record<string, boolean> = { ...selected }; pending.forEach((u) => next[u.id] = Boolean(v)); setSelected(next); }} /></TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Submitted</TableHead>
                  <TableHead>Reason</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pending.map((u) => (
                  <TableRow key={u.id} className={cn("odd:bg-foreground/[0.015] hover:bg-foreground/[0.03]", density === "compact" ? "h-11" : "h-14") }>
                    <TableCell className={cn("w-[40px]", density === "compact" && "py-1") }><Checkbox checked={!!selected[u.id]} onCheckedChange={(v) => setSelected((prev) => ({ ...prev, [u.id]: Boolean(v) }))} /></TableCell>
                    <TableCell className={cn(density === "compact" && "py-1")}>{u.name}</TableCell>
                    <TableCell className={cn(density === "compact" && "py-1")}>{u.email}</TableCell>
                    <TableCell className={cn(density === "compact" && "py-1")}>{u.requests?.[0]?.submittedAt ? formatDistanceToNow(new Date(u.requests[0].submittedAt), { addSuffix: true }) : "—"}</TableCell>
                    <TableCell className={cn("max-w-[280px] truncate", density === "compact" && "py-1")} title={u.requests?.[0]?.reason || ""}>{u.requests?.[0]?.reason || ""}</TableCell>
                    <TableCell className={cn("text-right", density === "compact" && "py-1")}>
                      <div className="flex justify-end gap-2">
                        <Button size="sm" className="hover:scale-[1.01]" onClick={() => approveOne(u)}>Approve</Button>
                        <Button size="sm" variant="outline" onClick={() => { setRejectTarget(u); setRejectReason(""); setNotify(true); setRejectOpen(true); }}>Reject</Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}

          {/* Reject confirm modal */}
          <AlertDialog open={rejectOpen}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Reject access request?</AlertDialogTitle>
                <AlertDialogDescription>Optionally include a reason. “Notify requester” is on by default.</AlertDialogDescription>
              </AlertDialogHeader>
              <div className="space-y-3">
                <div>
                  <label className="block text-sm mb-1">Reason (optional)</label>
                  <textarea className="w-full rounded-md border bg-background p-2 text-sm" rows={3} value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} />
                </div>
                <label className="flex items-center gap-2 text-sm"><Checkbox checked={notify} onCheckedChange={(v) => setNotify(Boolean(v))} /> Notify requester</label>
              </div>
              <AlertDialogFooter>
                <AlertDialogCancel onClick={() => setRejectOpen(false)}>Cancel</AlertDialogCancel>
                <AlertDialogAction onClick={() => {
                  if (rejectTarget) {
                    rejectOne(rejectTarget, rejectReason);
                  } else {
                    selectedIds.forEach((id) => rejectOne(users.find(u => u.id === id)!, rejectReason));
                  }
                  setRejectOpen(false);
                }}>Reject</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </>
      )}
    </div>
  );
}
