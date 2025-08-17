import { useEffect, useState } from "react";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
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
import { useToast } from "@/hooks/use-toast";
import { adminInviteCreate } from "@/lib/auth/admin";
import type { User } from "@/features/users/types";

type InviteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onInvited: (newUser: User) => void;
  defaultEmail?: string;
  defaultRole?: User["role"];
  requestId?: string;
  onAfterInvite?: (info: { requestId?: string; email: string }) => void;
};

const makeId = (prefix: string) => `${prefix}_${Math.random().toString(36).slice(2, 9)}`;

export default function InviteDialog({ open, onOpenChange, onInvited, defaultEmail, defaultRole = "member", requestId, onAfterInvite }: InviteDialogProps) {
  const { toast } = useToast();
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<User["role"]>("member");
  const [inviteDays, setInviteDays] = useState("7");
  const [inviteEmailUser, setInviteEmailUser] = useState(true);

  // Prefill when opening or when defaults change
  useEffect(() => {
    if (open) {
      if (defaultEmail) setInviteEmail(defaultEmail);
      setInviteRole(defaultRole || "member");
    }
  }, [open, defaultEmail, defaultRole]);

  const submitInvite = async () => {
    if (!inviteEmail) return;
    try {
      const out = await adminInviteCreate({
        email: inviteEmail,
        roles: [inviteRole],
        days_to_expire: parseInt(inviteDays) || 7,
        email_user: !!inviteEmailUser,
      });
      const link = out?.invite_link || "";
      const exp =
        out?.expires_at || new Date(Date.now() + (parseInt(inviteDays) || 7) * 86400e3).toISOString();
      const nowIso = new Date().toISOString();
      const newUser: User = {
        id: out?.invite_id || makeId("usr"),
        name:
          inviteEmail
            .split("@")[0]
            .replace(/\./g, " ")
            .replace(/\b\w/g, (c) => c.toUpperCase()) || "Pending User",
        email: inviteEmail,
        role: inviteRole,
        status: "pending_invite",
        createdAt: nowIso,
        lastActivityAt: undefined,
        sessions: 0,
        credentials: [],
        invites: { link, expiresAt: exp, lastSentAt: nowIso },
        requests: [],
      };
      onInvited(newUser);
      if (onAfterInvite) onAfterInvite({ requestId, email: inviteEmail });
      onOpenChange(false);
      setInviteEmail("");
      const message = `Invite sent to ${newUser.email}. Link copied to clipboard.`;
      toast({ title: message });
      if (link) {
        try {
          await navigator.clipboard.writeText(link);
        } catch {
          // ignore
        }
      }
    } catch {
      toast({ title: "Network error", variant: "destructive" });
    }
  };

  return (
    <AlertDialog open={open}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Invite user</AlertDialogTitle>
          <AlertDialogDescription>
            Send an invitation link with an optional expiration.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="space-y-3">
          <div>
            <label className="block text-sm mb-1">Email</label>
            <Input
              type="email"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              placeholder="user@company.com"
            />
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm mb-1">Role</label>
              <Select value={inviteRole} onValueChange={(v) => setInviteRole(v as "member" | "admin")}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="member">Member</SelectItem>
                  <SelectItem value="admin">Admin</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="block text-sm mb-1">Expiration</label>
              <Select value={inviteDays} onValueChange={setInviteDays}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="7">7 days</SelectItem>
                  <SelectItem value="14">14 days</SelectItem>
                  <SelectItem value="30">30 days</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="invite-email-user"
              checked={inviteEmailUser}
              onCheckedChange={(v) => setInviteEmailUser(Boolean(v))}
            />
            <label htmlFor="invite-email-user" className="text-sm select-none">
              Email user the invite link
            </label>
          </div>
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => onOpenChange(false)}>Cancel</AlertDialogCancel>
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
  );
}
