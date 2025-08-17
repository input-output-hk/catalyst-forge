import { useEffect, useMemo, useRef, useState, lazy, Suspense } from "react";
import { Helmet } from "react-helmet-async";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from "@/components/ui/tooltip";
import { useToast } from "@/hooks/use-toast";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { format, formatDistanceToNow } from "date-fns";
import {
  Download,
  Loader2,
  MoreVertical,
  Search,
  Trash2,
  Upload,
  Users,
  LogOut,
  Eye,
  Power,
  CheckCircle2,
  Slash,
  Mail,
  Clock,
  Rows3,
  List,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";
import { withLatency } from "@/mocks/latency";
import { forgeFetch } from "@/lib/client";
import { adminListUsers, adminDeleteUser, adminUpdateUser } from "@/lib/auth/admin";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";
import { useAppStore } from "@/store/app-store";
import type { User, Credential } from "@/features/users/types";
import Filters from "@/features/users/directory/components/Filters";
import BulkActionsBar from "@/features/users/directory/components/BulkActionsBar";
import UserTable from "@/features/users/directory/components/UserTable";
import PaginationControls from "@/features/users/directory/components/PaginationControls";

const RequestsTab = lazy(() => import("@/features/users/requests/RequestsTab"));
const InviteDialog = lazy(() => import("@/features/users/overlays/InviteDialog"));
const UserSheet = lazy(() => import("@/features/users/overlays/UserSheet"));

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
  const firstNames = [
    "Alice",
    "Bob",
    "Carol",
    "David",
    "Eve",
    "Frank",
    "Grace",
    "Heidi",
    "Ivan",
    "Judy",
    "Mallory",
    "Nia",
    "Olivia",
    "Peggy",
    "Rupert",
    "Sybil",
    "Trent",
    "Victor",
    "Walter",
    "Yara",
    "Zoe",
  ];
  const lastNames = [
    "Anderson",
    "Brown",
    "Clark",
    "Davis",
    "Evans",
    "Foster",
    "Garcia",
    "Harris",
    "Iverson",
    "Johnson",
    "Klein",
    "Lopez",
    "Miller",
    "Nguyen",
    "Olsen",
    "Patel",
    "Quinn",
    "Roberts",
    "Smith",
    "Turner",
    "Ulrich",
    "Vega",
    "Williams",
    "Xu",
    "Young",
    "Zimmerman",
  ];
  const roles: Array<User["role"]> = ["admin", "member"];
  const statuses: Array<User["status"]> = [
    "active",
    "disabled",
    "pending_invite",
    "pending_approval",
  ];
  const users: User[] = [];
  const now = Date.now();

  for (let i = 0; i < count; i++) {
    const fname = randomFrom(firstNames);
    const lname = randomFrom(lastNames);
    const name = `${fname} ${lname}`;
    const email = `${fname}.${lname}${i % 7 === 0 ? ".test" : ""}@example.com`.toLowerCase();
    const role = Math.random() < 0.18 ? "admin" : "member";
    const status = randomFrom(statuses);
    const createdAt = new Date(
      now - Math.floor(Math.random() * 1000 * 60 * 60 * 24 * 365)
    ).toISOString();
    const lastActivityAt =
      Math.random() < 0.12
        ? undefined
        : new Date(now - Math.floor(Math.random() * 1000 * 60 * 60 * 24 * 120)).toISOString();
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

    const invites =
      status === "pending_invite"
        ? {
            link: `https://app.example.com/invite/${makeId("inv")}`,
            expiresAt: new Date(now + 1000 * 60 * 60 * 24 * 7).toISOString(),
            lastSentAt: new Date(now - 1000 * 60 * 60 * 24).toISOString(),
          }
        : null;

    const requests =
      status === "pending_approval"
        ? [
            {
              submittedAt: new Date(
                now - 1000 * 60 * 60 * (Math.floor(Math.random() * 96) + 1)
              ).toISOString(),
              reason: Math.random() < 0.8 ? "Need access to run deployment workflows" : "",
              status: "pending" as const,
            },
          ]
        : [];

    users.push({
      id: makeId("usr"),
      name,
      email,
      role,
      status,
      createdAt,
      lastActivityAt,
      sessions,
      credentials,
      invites,
      requests,
    });
  }
  return users.sort((a, b) => a.name.localeCompare(b.name));
}

// Status and Role chips
function StatusBadge({ status }: { status: User["status"] }) {
  const map = {
    active: { label: "Active", tooltip: "Active", Icon: CheckCircle2, tone: "success" as const },
    disabled: {
      label: "Disabled",
      tooltip: "Disabled account",
      Icon: Slash,
      tone: "muted" as const,
    },
    pending_invite: {
      label: "Pending",
      tooltip: "Invite sent, awaiting acceptance",
      Icon: Mail,
      tone: "primary" as const,
    },
    pending_approval: {
      label: "Pending",
      tooltip: "Awaiting admin approval",
      Icon: Clock,
      tone: "info" as const,
    },
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
    <span
      aria-label={`Role: ${role}`}
      className="rounded-full border border-muted-foreground/60 px-2 py-0.5 text-xs text-muted-foreground"
    >
      {role === "admin" ? "Admin" : "Member"}
    </span>
  );
}

function HeaderUnderline() {
  return (
    <div className="mt-2 h-0.5 w-32 rounded-full bg-gradient-to-r from-primary via-primary/60 to-transparent" />
  );
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
  const [sort, setSort] = useState<{
    key: "name" | "lastActivityAt" | "createdAt";
    dir: "asc" | "desc";
  }>({ key: "name", dir: "asc" });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [serverTotal, setServerTotal] = useState<number | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const requestsSearchRef = useRef<HTMLInputElement>(null);
  const [activeTab, setActiveTab] = useState<"directory" | "requests">("directory");
  const { toast } = useToast();
  const { state } = useAppStore();
  const currentUserEmail = state.session.user || "";

  // User sheet loads its own data lazily

  // Load users from API with filters/paging (fallback to mock seed on error for now)
  useEffect(() => {
    (async () => {
      setLoading(true);
      try {
        const offset = (page - 1) * pageSize;
        const { data, total } = await adminListUsers({
          q: search.trim() || undefined,
          role: roleFilter !== "all" ? roleFilter : undefined,
          limit: pageSize,
          offset,
        });
        if (typeof total === "number") setServerTotal(total);
        const mapped: User[] = data.users.map((u) => ({
          id: u.id,
          name: u.email,
          email: u.email,
          role: (u.roles || []).includes("admin") ? "admin" : "member",
          status: "active",
          createdAt: u.created_at,
          lastActivityAt: u.last_activity_at || undefined,
          sessions: (u as unknown as { active_sessions?: number }).active_sessions ?? 0,
          credentials: [],
          invites: null,
          requests: [],
        }));
        setUsers(mapped);
      } catch {
        await withLatency(350, 700);
        if (import.meta.env.DEV) {
          const mod = await import("@/features/users/utils/seed");
          setUsers(mod.seedUsers(120));
        } else {
          setUsers([]);
        }
        setServerTotal(null);
      } finally {
        setLoading(false);
      }
    })();
  }, [search, roleFilter, page, pageSize]);

  // Persist density preference
  useEffect(() => {
    const saved = localStorage.getItem("users:density");
    if (saved === "comfortable" || saved === "compact")
      setDensity(saved as "comfortable" | "compact");
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
      if (e.shiftKey && e.key.toLowerCase() === "a") {
        e.preventDefault();
        setInviteOpen(true);
      }
      if (e.shiftKey && e.key.toLowerCase() === "d") {
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
    let list = users.filter(
      (u) =>
        !s ||
        u.name.toLowerCase().includes(s) ||
        u.email.toLowerCase().includes(s) ||
        u.id.toLowerCase().includes(s)
    );

    if (statusFilter !== "all") list = list.filter((u) => u.status === statusFilter);
    if (roleFilter !== "all") list = list.filter((u) => u.role === roleFilter);
    if (activityFilter !== "all") {
      const now = Date.now();
      const ranges: Record<string, number> = {
        "24h": 24,
        "7d": 24 * 7,
        "30d": 24 * 30,
        "90d": 24 * 90,
      };
      if (activityFilter === "never") {
        list = list.filter((u) => !u.lastActivityAt);
      } else if (ranges[activityFilter]) {
        const hours = ranges[activityFilter];
        const cutoff = now - hours * 60 * 60 * 1000;
        list = list.filter((u) =>
          u.lastActivityAt ? new Date(u.lastActivityAt).getTime() >= cutoff : false
        );
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
      if (sort.key === "createdAt")
        res = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
      return sort.dir === "asc" ? res : -res;
    });

    return sorted;
  }, [users, search, statusFilter, roleFilter, activityFilter, sort]);

  const total = serverTotal ?? filtered.length;
  const totalAll = serverTotal ?? users.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const startIdx = (page - 1) * pageSize;
  const endIdx = Math.min(startIdx + pageSize, total);
  const visible = serverTotal != null ? users : filtered.slice(startIdx, endIdx);
  const hiddenCount = Math.max(0, (serverTotal != null ? serverTotal : users.length) - total);
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

  // Actions
  const bulkDisable = () => {
    setUsers((prev) =>
      prev.map((u) => (selected[u.id] ? { ...u, status: "disabled", sessions: 0 } : u))
    );
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
    (async () => {
      const ids = Object.entries(selected)
        .filter(([, v]) => v)
        .map(([id]) => id);
      for (const id of ids) {
        try {
          await adminDeleteUser(id);
        } catch (e) {
          toast({
            title: `Failed deleting ${id}`,
            description: (e as Error).message,
            variant: "destructive",
          });
        }
      }
      setUsers((prev) => prev.filter((u) => !selected[u.id]));
      toast({ title: `${ids.length} users deleted.` });
      setSelected({});
    })();
  };
  const changeRoleBulk = (role: User["role"]) => {
    setUsers((prev) => prev.map((u) => (selected[u.id] ? { ...u, role } : u)));
    toast({ title: `Changed role to ${role} for ${selectedIds.length}.` });
    setSelected({});
  };

  // CSV Export
  const exportCsv = async () => {
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
    const mod = await import("@/features/users/utils/csv");
    mod.exportUsersCsv(rows as Array<Record<string, unknown>>);
  };

  // Invite dialog visibility
  const [inviteOpen, setInviteOpen] = useState(false);
  useEffect(() => {
    const openHandler = () => setInviteOpen(true);
    window.addEventListener("cf:open-invite", openHandler as EventListener);
    return () => window.removeEventListener("cf:open-invite", openHandler as EventListener);
  }, []);

  // User selected for sheet
  const openUser = users.find((u) => u.id === openUserId) || null;

  return (
    <div className="p-4 md:p-6">
      <Helmet>
        <title>Users | Admin Console</title>
        <meta
          name="description"
          content="Manage and audit users: search, filter, view details, and take actions."
        />
        <link rel="canonical" href="/users" />
      </Helmet>

      <header className="mb-4 md:mb-6">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl md:text-3xl font-semibold tracking-tight">
              Users{" "}
              <span className="ml-2 align-middle text-xs font-normal text-muted-foreground">
                {totalAll.toLocaleString()}
              </span>
            </h1>
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

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as "directory" | "requests")}>
        <div className="flex items-center gap-3 flex-wrap">
          <TabsList>
            <TabsTrigger value="directory">Directory</TabsTrigger>
            <TabsTrigger value="requests">Requests</TabsTrigger>
          </TabsList>
          <div className="hidden sm:block h-6 w-px bg-border/60" aria-hidden />
          <ToggleGroup
            type="single"
            value={density}
            onValueChange={(v) => v && setDensity(v as "comfortable" | "compact")}
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
                <Input
                  ref={searchRef}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search name, email, ID ( / )"
                  className="pl-8"
                  aria-label="Search users"
                />
              </div>
            </div>
            <Filters
              statusFilter={statusFilter}
              roleFilter={roleFilter}
              activityFilter={activityFilter}
              setStatusFilter={setStatusFilter}
              setRoleFilter={setRoleFilter}
              setActivityFilter={setActivityFilter}
            />
          </div>

          {/* Active filter chips */}
          {(statusFilter !== "all" || roleFilter !== "all" || activityFilter !== "all") && (
            <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
              {statusFilter !== "all" && (
                <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
                  <span className="text-muted-foreground">
                    Status: {statusFilter.replace("_", " ")}
                  </span>
                  <button
                    className="hover:text-foreground"
                    onClick={() => setStatusFilter("all")}
                    aria-label="Clear status"
                  >
                    ×
                  </button>
                </span>
              )}
              {roleFilter !== "all" && (
                <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
                  <span className="text-muted-foreground">Role: {roleFilter}</span>
                  <button
                    className="hover:text-foreground"
                    onClick={() => setRoleFilter("all")}
                    aria-label="Clear role"
                  >
                    ×
                  </button>
                </span>
              )}
              {activityFilter !== "all" && (
                <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/50 px-2 py-0.5">
                  <span className="text-muted-foreground">Last activity: {activityFilter}</span>
                  <button
                    className="hover:text-foreground"
                    onClick={() => setActivityFilter("all")}
                    aria-label="Clear last activity"
                  >
                    ×
                  </button>
                </span>
              )}
              <button
                className="ml-auto text-muted-foreground hover:text-foreground"
                onClick={() => {
                  setStatusFilter("all");
                  setRoleFilter("all");
                  setActivityFilter("all");
                }}
              >
                Clear all
              </button>
            </div>
          )}

          {/* Selection bar */}
          {selectedIds.length > 0 && (
            <BulkActionsBar
              selectedCount={selectedIds.length}
              onDisable={bulkDisable}
              onEnable={bulkEnable}
              onLogout={bulkLogout}
              onChangeRole={changeRoleBulk}
              onDelete={bulkDelete}
            />
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
                primaryCta={{
                  label: "Clear filters",
                  onClick: () => {
                    setSearch("");
                    setStatusFilter("all");
                    setRoleFilter("all");
                    setActivityFilter("all");
                  },
                }}
                secondaryCta={{ label: "Invite user", onClick: () => setInviteOpen(true) }}
              />
            ) : (
              <UserTable
                users={users}
                density={density}
                currentUserEmail={currentUserEmail}
                visible={visible}
                selected={selected}
                onSelectUser={(id, checked) => setSelected((prev) => ({ ...prev, [id]: checked }))}
                onToggleAllVisible={(checked) => toggleAllVisible(checked)}
                allVisibleSelected={allVisibleSelected}
                onOpenUser={(id) => setOpenUserId(id)}
                onDisableToggle={async (u) => {
                  try {
                    const res = await forgeFetch(`/api/v1/admin/users/${u.id}`, {
                      method: "PATCH",
                      headers: { "content-type": "application/json" },
                      body: JSON.stringify({ suspend: u.status !== "disabled" }),
                    });
                    if (res.status === 204) {
                      setUsers((prev) =>
                        prev.map((x) =>
                          x.id === u.id
                            ? {
                                ...x,
                                status: u.status !== "disabled" ? "disabled" : "active",
                                sessions: u.status !== "disabled" ? 0 : x.sessions,
                              }
                            : x
                        )
                      );
                      toast({
                        title: u.status !== "disabled" ? "User disabled" : "User re-enabled",
                      });
                    } else {
                      const text = await res.text().catch(() => "");
                      toast({ title: "Failed", description: text || `${res.status}` });
                    }
                  } catch {
                    toast({ title: "Network error" });
                  }
                }}
                onLogout={(id) =>
                  setUsers((prev) => prev.map((x) => (x.id === id ? { ...x, sessions: 0 } : x)))
                }
                onDelete={(id) => setUsers((prev) => prev.filter((x) => x.id !== id))}
                sortKey={sort.key}
                sortDir={sort.dir}
                onChangeSort={(key) =>
                  setSort((s) => ({ key, dir: s.key === key && s.dir === "asc" ? "desc" : "asc" }))
                }
              />
            )}
          </div>

          {/* Pagination */}
          <PaginationControls
            loading={loading}
            total={total}
            page={page}
            pageCount={pageCount}
            pageSize={pageSize}
            setPage={setPage}
            setPageSize={setPageSize}
            startIdx={startIdx}
            endIdx={endIdx}
          />
        </TabsContent>

        {/* Requests Tab (lazy) */}
        <TabsContent value="requests" className="mt-4">
          <Suspense fallback={null}>
            <RequestsTab
              users={users}
              setUsers={setUsers}
              density={density}
              setDensity={setDensity}
              searchRef={requestsSearchRef}
            />
          </Suspense>
        </TabsContent>
      </Tabs>

      {/* Invite dialog */}
      <Suspense fallback={null}>
        {inviteOpen && (
          <InviteDialog
            open={inviteOpen}
            onOpenChange={setInviteOpen}
            onInvited={(newUser) => setUsers((prev) => [newUser, ...prev])}
          />
        )}
      </Suspense>

      {/* User slide-over */}
      <Suspense fallback={null}>
        {openUser && (
          <UserSheet
            user={openUser}
            onOpenChange={(o) => setOpenUserId(o ? openUserId : null)}
            density={density}
          />
        )}
      </Suspense>
    </div>
  );
}

function RowActions({
  user,
  onOpen,
  onChange,
  onDelete,
}: {
  user: User;
  onOpen: () => void;
  onChange: (u: User) => void;
  onDelete: () => void;
}) {
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
          <DropdownMenuItem onClick={() => onChange({ ...user, status: "active" })}>
            Re-enable
          </DropdownMenuItem>
        ) : (
          <DropdownMenuItem onClick={() => onChange({ ...user, status: "disabled", sessions: 0 })}>
            Disable
          </DropdownMenuItem>
        )}
        <DropdownMenuItem onClick={() => onChange({ ...user, sessions: 0 })}>
          Force log out
        </DropdownMenuItem>
        {user.status === "pending_invite" && (
          <DropdownMenuItem onClick={() => toast({ title: `Invite resent to ${user.email}.` })}>
            Resend invite
          </DropdownMenuItem>
        )}
        <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={onDelete}>
          <Trash2 className="h-4 w-4 mr-2" /> Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function Confirm({
  label,
  description,
  destructive,
  onConfirm,
}: {
  label: string;
  description?: string;
  destructive?: boolean;
  onConfirm: () => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="sm"
        variant={destructive ? "destructive" : "secondary"}
        onClick={() => setOpen(true)}
      >
        {label}
      </Button>
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="rounded-md border bg-background p-4 shadow-md w-[360px] max-w-[92vw]">
            <div className="mb-2 text-sm font-medium">{label}</div>
            {description && <div className="mb-3 text-sm text-muted-foreground">{description}</div>}
            <div className="flex justify-end gap-2">
              <Button size="sm" variant="outline" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button
                size="sm"
                variant={destructive ? "destructive" : "default"}
                onClick={() => {
                  onConfirm();
                  setOpen(false);
                }}
              >
                Confirm
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

// Requests tab moved to @/features/users/requests/RequestsTab
