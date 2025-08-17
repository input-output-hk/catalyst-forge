import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

type Props = {
  statusFilter: string;
  roleFilter: string;
  activityFilter: string;
  setStatusFilter: (v: string) => void;
  setRoleFilter: (v: string) => void;
  setActivityFilter: (v: string) => void;
};

export default function Filters({
  statusFilter,
  roleFilter,
  activityFilter,
  setStatusFilter,
  setRoleFilter,
  setActivityFilter,
}: Props) {
  return (
    <div className="flex items-center gap-2">
      <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v)}>
        <SelectTrigger className="w-[160px]">
          <SelectValue placeholder="Status" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Status: All</SelectItem>
          <SelectItem value="active">Active</SelectItem>
          <SelectItem value="disabled">Disabled</SelectItem>
          <SelectItem value="pending_invite">Pending invite</SelectItem>
          <SelectItem value="pending_approval">Pending approval</SelectItem>
        </SelectContent>
      </Select>
      <Select value={roleFilter} onValueChange={(v) => setRoleFilter(v)}>
        <SelectTrigger className="w-[140px]">
          <SelectValue placeholder="Role" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Role: All</SelectItem>
          <SelectItem value="admin">Admin</SelectItem>
          <SelectItem value="member">Member</SelectItem>
        </SelectContent>
      </Select>
      <Select value={activityFilter} onValueChange={setActivityFilter}>
        <SelectTrigger className="w-[160px]">
          <SelectValue placeholder="Last activity" />
        </SelectTrigger>
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
  );
}
