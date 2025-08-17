import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import type { User } from "@/features/users/types";
import UserRow from "./UserRow";

type Props = {
  users: User[];
  density: "comfortable" | "compact";
  currentUserEmail: string;
  visible: User[];
  selected: Record<string, boolean>;
  onSelectUser: (id: string, checked: boolean) => void;
  onToggleAllVisible: (checked: boolean) => void;
  allVisibleSelected: boolean;
  onOpenUser: (id: string) => void;
  onDisableToggle: (user: User) => Promise<void> | void;
  onLogout: (id: string) => void;
  onDelete: (id: string) => void;
  sortKey: "name" | "lastActivityAt" | "createdAt";
  sortDir: "asc" | "desc";
  onChangeSort: (key: "name" | "lastActivityAt" | "createdAt") => void;
};

export default function UserTable({
  density,
  currentUserEmail,
  visible,
  selected,
  onSelectUser,
  onToggleAllVisible,
  allVisibleSelected,
  onOpenUser,
  onDisableToggle,
  onLogout,
  onDelete,
  sortKey,
  sortDir,
  onChangeSort,
}: Props) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[40px]">
            <Checkbox
              checked={allVisibleSelected}
              onCheckedChange={(v) => onToggleAllVisible(Boolean(v))}
              aria-label="Select all visible"
            />
          </TableHead>
          <TableHead className="cursor-pointer select-none" onClick={() => onChangeSort("name")}>
            User {sortKey === "name" && (sortDir === "asc" ? "↑" : "↓")}
          </TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Role</TableHead>
          <TableHead
            className="cursor-pointer select-none"
            onClick={() => onChangeSort("lastActivityAt")}
          >
            Last activity {sortKey === "lastActivityAt" && (sortDir === "asc" ? "↑" : "↓")}
          </TableHead>
          <TableHead>Sessions</TableHead>
          <TableHead
            className="cursor-pointer select-none"
            onClick={() => onChangeSort("createdAt")}
          >
            Created {sortKey === "createdAt" && (sortDir === "asc" ? "↑" : "↓")}
          </TableHead>
          <TableHead className="w-[120px] text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {visible.map((u) => (
          <UserRow
            key={u.id}
            user={u}
            density={density}
            currentUserEmail={currentUserEmail}
            selected={!!selected[u.id]}
            onSelect={(checked) => onSelectUser(u.id, checked)}
            onOpen={onOpenUser}
            onDisableToggle={onDisableToggle}
            onLogout={onLogout}
            onDelete={onDelete}
          />
        ))}
      </TableBody>
    </Table>
  );
}
