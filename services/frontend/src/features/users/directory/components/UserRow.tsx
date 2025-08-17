import { format, formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Eye, LogOut, MoreVertical, Power, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { User } from "@/features/users/types";

type Props = {
  user: User;
  density: "comfortable" | "compact";
  currentUserEmail: string;
  selected: boolean;
  onSelect: (checked: boolean) => void;
  onOpen: (id: string) => void;
  onDisableToggle: (user: User) => Promise<void> | void;
  onLogout: (id: string) => void;
  onDelete: (id: string) => void;
  onResendInvite?: (id: string) => void;
};

function firstLast(name: string) {
  const parts = name.split(" ");
  const first = parts[0]?.[0] ?? "?";
  const last = parts[1]?.[0] ?? parts[0]?.[1] ?? "";
  return (first + last).toUpperCase();
}

function RoleBadge({ role }: { role: User["role"] }) {
  return (
    <span className="rounded-full border border-muted-foreground/60 px-2 py-0.5 text-xs text-muted-foreground">
      {role === "admin" ? "Admin" : "Member"}
    </span>
  );
}

function StatusBadge({ status }: { status: User["status"] }) {
  const label = status === "active" ? "Active" : status === "disabled" ? "Disabled" : "Pending";
  return (
    <span className="inline-flex items-center justify-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 min-w-[96px] bg-muted text-muted-foreground ring-border/30">
      <span>{label}</span>
    </span>
  );
}

export default function UserRow({
  user: u,
  density,
  currentUserEmail,
  selected,
  onSelect,
  onOpen,
  onDisableToggle,
  onLogout,
  onDelete,
  onResendInvite,
}: Props) {
  return (
    <tr
      className={cn(
        "cursor-pointer group",
        density === "compact" ? "h-11" : "h-14",
        "odd:bg-foreground/[0.015] even:bg-transparent hover:bg-foreground/[0.03]"
      )}
      onClick={(e) => {
        const tag = (e.target as HTMLElement).closest("button,input,svg,span,[role='menuitem']");
        if (tag) return;
        onOpen(u.id);
      }}
    >
      <td
        className={cn("w-[40px]", density === "compact" && "py-1")}
        onClick={(e) => e.stopPropagation()}
      >
        <Checkbox
          checked={selected}
          onCheckedChange={(v) => onSelect(Boolean(v))}
          aria-label={`Select ${u.name}`}
        />
      </td>
      <td className={cn(density === "compact" && "py-1")}>
        <div className="flex items-center gap-3">
          <Avatar className="h-9 w-9">
            <AvatarFallback>{firstLast(u.name)}</AvatarFallback>
          </Avatar>
          <div>
            <div className={cn("font-medium", density === "compact" ? "text-sm" : "")}>
              {u.name && u.name.trim() && u.name.trim().toLowerCase() !== u.email.toLowerCase()
                ? u.name
                : u.email}
            </div>
            {u.name && u.name.trim() && u.name.trim().toLowerCase() !== u.email.toLowerCase() && (
              <div className="text-xs text-muted-foreground">{u.email}</div>
            )}
          </div>
        </div>
      </td>
      <td className={cn(density === "compact" && "py-1")}>
        <StatusBadge status={u.status} />
      </td>
      <td className={cn(density === "compact" && "py-1")}>
        <RoleBadge role={u.role} />
      </td>
      <td
        className={cn(density === "compact" && "py-1")}
        title={u.lastActivityAt ? format(new Date(u.lastActivityAt), "PPpp") : "Never"}
      >
        {u.lastActivityAt
          ? `${formatDistanceToNow(new Date(u.lastActivityAt), { addSuffix: true })}`
          : "Never"}
      </td>
      <td className={cn(density === "compact" && "py-1")}>{u.sessions}</td>
      <td
        className={cn(density === "compact" && "py-1")}
        title={format(new Date(u.createdAt), "PPpp")}
      >
        {format(new Date(u.createdAt), "PP")}
      </td>
      <td
        className={cn("text-right", density === "compact" && "py-1")}
        onClick={(e) => e.stopPropagation()}
      >
        <TooltipProvider>
          <div className="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" aria-label="View" onClick={() => onOpen(u.id)}>
                  <Eye className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>View</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={u.status === "disabled" ? "Re-enable" : "Disable"}
                  disabled={u.email === currentUserEmail || u.role === "admin"}
                  onClick={() => onDisableToggle(u)}
                >
                  <Power className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                {u.email === currentUserEmail
                  ? "You can’t disable yourself"
                  : u.role === "admin"
                    ? "Admins cannot be disabled here"
                    : u.status === "disabled"
                      ? "Re-enable"
                      : "Disable"}
              </TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Force log out"
                  onClick={() => onLogout(u.id)}
                >
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
                  <DropdownMenuItem onClick={() => onResendInvite?.(u.id)}>
                    Resend invite
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  className="text-destructive focus:text-destructive"
                  onClick={() => onDelete(u.id)}
                >
                  <Trash2 className="h-4 w-4 mr-2" /> Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </TooltipProvider>
      </td>
    </tr>
  );
}
