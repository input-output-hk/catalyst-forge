import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ProfileEditDialog } from "@/components/auth/ProfileEditDialog";
import { useAppStore } from "@/store/app-store";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";
import {
  Key,
  Monitor,
  RotateCcw,
  Trash2,
  MoreVertical,
  Edit,
  LogOut,
  Info,
  Terminal,
  Check,
} from "lucide-react";
import { format, formatDistanceToNow } from "date-fns";
import { generateRecoveryKeys } from "@/lib/auth/recovery";
import { extractErrorMessage, readResponseError } from "@/lib/api";
import { forge } from "@/lib/client";
import { Skeleton } from "@/components/ui/skeleton";
import type { components } from "forge-client";
import { listCredentials, renameCredential, deleteCredential } from "@/lib/auth/credentials";
import {
  listSessions,
  deleteSession,
  deleteSessions,
  logoutAllSessions,
} from "@/lib/auth/sessions";
import { updateFullName } from "@/lib/auth/profile";
import { regenerateRecoveryCodes } from "@/lib/auth/recovery";

// Type definitions
type RecoveryGenerateResponse = components["schemas"]["auth.RecoveryGenerateResponse"];
type SessionsListResponse = components["schemas"]["auth.SessionsListResponse"];
type CredentialsListResponse = components["schemas"]["auth.CredentialsListResponse"];

interface UISession {
  id: string;
  browser: string;
  os: string;
  location: string;
  lastActivity: string;
  loginDate: string;
  current: boolean;
  userAgent?: string;
  ipAddress?: string;
  amr?: string;
  lastActivityAtMs: number;
}

/**
 * Profile page component for managing user account settings.
 * Handles passkeys, sessions, recovery keys, and account management.
 */
const Profile = () => {
  const { state, actions } = useAppStore();
  const { toast } = useToast();
  const helmet = usePageTitle(
    "My Profile – Catalyst Forge",
    "Manage your account information, devices, and security.",
    "/profile"
  );

  const [fullName, setFullName] = useState(
    state.session.user ? state.session.user.split("@")[0].replace(".", " ") : ""
  );
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [sessions, setSessions] = useState<UISession[]>([]);
  type CredentialItem = components["schemas"]["auth.CredentialSummary"];
  const [credentials, setCredentials] = useState<CredentialItem[]>([]);
  const [loadingCreds, setLoadingCreds] = useState(false);
  const [recoveryKeysGenerated, setRecoveryKeysGenerated] = useState(true);
  const [lastRecoveryDate] = useState(new Date(Date.now() - 1000 * 60 * 60 * 24 * 30));

  // Rename device dialog state
  const [renameOpen, setRenameOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<CredentialItem | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [copiedSessionId, setCopiedSessionId] = useState<string | null>(null);

  const userInitials = state.session.user
    ? state.session.user
        .split("@")[0]
        .split(".")
        .map((part) => part[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "U";

  /**
   * Handles saving profile name changes.
   * @param newName - The new display name for the user
   */
  const handleSave = async (newName: string) => {
    try {
      await updateFullName(newName);
      {
        setFullName(newName);
        toast({ title: "Profile updated successfully" });
      }
    } catch {
      toast({
        title: "Failed to update profile",
        variant: "destructive",
      });
    }
  };

  /**
   * Refreshes the list of user credentials (passkeys).
   */
  const refreshCredentials = useCallback(async () => {
    setLoadingCreds(true);

    try {
      const data = await listCredentials();
      const credentialsList = data?.credentials || [];
      const typedCredentials = credentialsList as unknown as CredentialItem[];
      setCredentials(typedCredentials);
    } finally {
      setLoadingCreds(false);
    }
  }, []);

  useEffect(() => {
    void refreshCredentials();
  }, [refreshCredentials]);

  /**
   * Logs out all devices associated with the user account.
   */
  const handleLogoutAllDevices = async () => {
    try {
      await logoutAllSessions();

      toast({ title: "Logged out of all devices" });

      // Refresh both credentials and sessions
      void refreshCredentials();
      void refreshSessions();
    } catch {
      toast({ title: "Failed to log out of all devices" });
    }
  };

  /**
   * Regenerates recovery keys for account recovery.
   */
  const handleRegenerateKeys = async () => {
    try {
      const codes = await regenerateRecoveryCodes();
      actions.startRecoveryGate(codes, "/profile");
      setRecoveryKeysGenerated(true);

      toast({
        title: "Recovery keys regenerated",
        description: "Make sure to save them securely.",
      });
    } catch (error) {
      const errorMessage = extractErrorMessage(error);
      toast({
        title: "Failed to regenerate recovery keys",
        description: errorMessage,
        variant: "destructive",
      });
    }
  };

  const handleEndSession = async (sessionId: string) => {
    try {
      await deleteSession(sessionId);
      {
        toast({ title: "Session terminated" });
      }
    } catch {
      toast({ title: "Failed to terminate session", variant: "destructive" });
    } finally {
      void refreshSessions();
    }
  };

  const handleTerminateAllSessions = async () => {
    try {
      const targets = sessions.filter((s) => !s.current).map((s) => s.id);
      await deleteSessions(targets);
      toast({ title: "All other sessions terminated" });
    } catch {
      toast({ title: "Failed to terminate all other sessions", variant: "destructive" });
    } finally {
      void refreshSessions();
    }
  };

  /**
   * Refreshes the list of active sessions.
   */
  const refreshSessions = useCallback(async () => {
    try {
      const data = await listSessions();
      const sessionItems = data?.sessions || [];

      // Map sessions to UI format
      const mappedSessions: UISession[] = sessionItems.map((session) => {
        const isCliDevice = session.amr === "device_link";
        const deviceName = session.device_name || "";
        const cliLabel = `CLI device${deviceName ? ` • ${deviceName}` : ""}`;

        const browserLabel = isCliDevice
          ? cliLabel
          : session.current
            ? "This browser"
            : "Passkey session";

        const lastActivityTime = session.last_activity_at
          ? new Date(session.last_activity_at).getTime()
          : 0;

        return {
          id: session.id,
          browser: browserLabel,
          amr: session.amr,
          os: "",
          location: "",
          lastActivity: formatLastUsedLabel(session.last_activity_at),
          loginDate: session.created_at,
          current: !!session.current,
          userAgent: session.user_agent || "",
          ipAddress: session.ip_address || "",
          lastActivityAtMs: lastActivityTime,
        };
      });

      // Sort sessions: current first, then by activity
      mappedSessions.sort((a, b) => {
        const currentDifference = Number(b.current) - Number(a.current);

        if (currentDifference !== 0) {
          return currentDifference;
        }

        return b.lastActivityAtMs - a.lastActivityAtMs;
      });

      setSessions(mappedSessions);
    } catch {
      // Silently ignore errors
    }
  }, []);

  // Load active sessions after callbacks are defined
  useEffect(() => {
    void refreshSessions();
  }, [refreshSessions]);

  const handleDeleteAccount = () => {
    toast({
      title: "Account deletion initiated",
      description: "Your account will be deleted within 24 hours.",
    });
  };

  const openRenameDialog = (cred: CredentialItem) => {
    setRenameTarget(cred);
    setRenameValue(cred.device_name ?? "");
    setRenameOpen(true);
  };

  const formatLastUsedLabel = (iso?: string | null): string => {
    if (!iso) return "—";
    const d = new Date(iso);
    const ms = Date.now() - d.getTime();
    if (ms < 60 * 1000) return "Just now";
    if (ms < 7 * 24 * 60 * 60 * 1000) {
      return formatDistanceToNow(d, { addSuffix: true });
    }
    return format(d, "PP");
  };

  const isRecent = (iso?: string | null): boolean => {
    if (!iso) return false;
    return Date.now() - new Date(iso).getTime() < 24 * 60 * 60 * 1000;
  };

  const submitRename = async () => {
    const target = renameTarget;
    const newName = renameValue.trim();
    if (!target || newName.length === 0 || newName === target.device_name) {
      setRenameOpen(false);
      return;
    }
    try {
      await renameCredential(target.id!, newName);
      {
        toast({ title: "Device renamed" });
        setRenameOpen(false);
        setRenameTarget(null);
        setRenameValue("");
        void refreshCredentials();
      }
    } catch {
      toast({ title: "Failed to rename device", variant: "destructive" });
    }
  };

  return (
    <div className="container mx-auto py-6 max-w-6xl">
      {helmet}

      <div className="space-y-8">
        {/* Header */}
        <section className="flex items-center justify-between gap-4 pb-4 border-b border-border/50">
          <div className="flex items-center gap-4 min-w-0">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Avatar
                    className="h-12 w-12 shrink-0 cursor-pointer hover:opacity-80 transition-opacity rounded-full focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    role="button"
                    tabIndex={0}
                    aria-label="Edit profile"
                    onClick={() => setIsEditDialogOpen(true)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        setIsEditDialogOpen(true);
                      }
                    }}
                  >
                    <AvatarFallback className="bg-primary/10 text-primary font-medium">
                      {userInitials}
                    </AvatarFallback>
                  </Avatar>
                </TooltipTrigger>
                <TooltipContent>
                  <p>Click to change avatar</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h1 className="truncate text-xl font-semibold">{fullName || state.session.user}</h1>
                {Array.isArray(state.session.roles) && state.session.roles.includes("admin") ? (
                  <span className="rounded-full bg-muted px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground">
                    Administrator
                  </span>
                ) : (
                  <span className="rounded-full bg-muted px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground">
                    Member
                  </span>
                )}
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setIsEditDialogOpen(true)}
                  className="text-muted-foreground hover:text-foreground p-1 h-auto"
                  aria-label="Edit profile name"
                >
                  <Edit className="h-3 w-3" />
                </Button>
              </div>
              <div className="truncate text-sm text-muted-foreground">{state.session.user}</div>
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleLogoutAllDevices} className="text-destructive">
                <LogOut className="h-4 w-4 mr-2" />
                Log out of all devices
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleDeleteAccount} className="text-destructive">
                <Trash2 className="h-4 w-4 mr-2" />
                Delete account
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </section>

        {/* Grid */}
        <div className="mt-16 grid grid-cols-1 gap-6 lg:grid-cols-2">
          {/* Passkeys & Security Keys */}
          <section className="space-y-4">
            <div className="flex items-baseline justify-between flex-wrap gap-2">
              <h2 className="text-xl font-bold">Passkeys & Security Keys</h2>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  onClick={() =>
                    toast({ title: "Register a new passkey from your browser (coming soon)" })
                  }
                >
                  + Register passkey
                </Button>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button size="sm" variant="outline" className="text-destructive">
                      Log out of all sessions
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Log out of all sessions?</AlertDialogTitle>
                      <AlertDialogDescription>
                        This will sign you out everywhere. You'll need to sign in again.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction
                        onClick={handleLogoutAllDevices}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                      >
                        Log out all
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </div>
            <ul className="divide-y divide-border/40 rounded-lg border border-border/40">
              {loadingCreds ? (
                <>
                  {[0, 1, 2].map((i) => (
                    <li key={i} className="py-3 px-3">
                      <div className="flex items-center gap-3">
                        <Skeleton className="h-4 w-4 rounded-full" />
                        <div className="min-w-0 flex-1 space-y-1.5">
                          <Skeleton className="h-4 w-40" />
                          <Skeleton className="h-3 w-64" />
                        </div>
                      </div>
                    </li>
                  ))}
                </>
              ) : credentials.length === 0 ? (
                <li className="py-6 px-3 text-sm text-muted-foreground">No passkeys yet.</li>
              ) : (
                [...credentials]
                  .sort((a, b) => {
                    const timeA = a.last_used_at ? new Date(a.last_used_at).getTime() : 0;
                    const timeB = b.last_used_at ? new Date(b.last_used_at).getTime() : 0;

                    return timeB - timeA;
                  })
                  .map((cred) => (
                    <li
                      key={cred.id}
                      className="flex items-center justify-between py-3.5 px-3 hover:bg-accent/40 rounded-md transition-colors"
                    >
                      <div className="min-w-0 flex items-center gap-3">
                        <Key className="h-4 w-4 text-muted-foreground shrink-0" />
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <div className="font-medium truncate">{cred.device_name}</div>
                            {isRecent(cred.last_used_at) && (
                              <span className="rounded-full bg-info/15 text-info px-2 py-0.5 text-[10px]">
                                Recent
                              </span>
                            )}
                          </div>
                          <div className="text-xs text-muted-foreground space-y-0.5 leading-5">
                            <div>Sign count {cred.sign_count}</div>
                            <div>Last used {formatLastUsedLabel(cred.last_used_at)}</div>
                          </div>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="ghost"
                          className="text-foreground/80 hover:text-foreground"
                          aria-label={`Rename ${cred.device_name}`}
                          onClick={() => openRenameDialog(cred)}
                        >
                          Rename
                        </Button>
                        {credentials.length <= 1 ? (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="inline-flex">
                                  <Button
                                    size="sm"
                                    variant="ghost"
                                    className="text-destructive"
                                    aria-label={`Delete ${cred.device_name}`}
                                    disabled
                                  >
                                    Delete
                                  </Button>
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>You must keep at least one passkey</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        ) : (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button
                                size="sm"
                                variant="ghost"
                                className="text-destructive hover:text-destructive hover:bg-destructive/5"
                                aria-label={`Delete ${cred.device_name}`}
                              >
                                Delete
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Remove device?</AlertDialogTitle>
                                <AlertDialogDescription>
                                  This will remove "{cred.device_name}" from your account. You must
                                  have at least one passkey; the last one cannot be removed.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={async () => {
                                    try {
                                      await deleteCredential(cred.id!);
                                      toast({ title: "Passkey removed" });
                                    } catch {
                                      toast({ title: "Failed to remove passkey" });
                                    } finally {
                                      void refreshCredentials();
                                    }
                                  }}
                                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                >
                                  Remove
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </div>
                    </li>
                  ))
              )}
            </ul>
          </section>

          {/* Active Sessions */}
          <section className="space-y-4">
            <div className="flex items-baseline justify-between flex-wrap gap-2">
              <h2 className="text-xl font-bold">Active Sessions</h2>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    size="sm"
                    variant="outline"
                    className="whitespace-nowrap text-destructive"
                  >
                    Terminate all sessions
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Terminate all sessions?</AlertDialogTitle>
                    <AlertDialogDescription>
                      This will end all other active sessions except your current one. Other devices
                      will need to sign in again.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                      onClick={handleTerminateAllSessions}
                      className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                      Terminate all
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
            <ul className="divide-y divide-border/40 rounded-lg border border-border/40">
              {sessions.length === 0 ? (
                <li className="py-6 px-3 text-sm text-muted-foreground">No active sessions.</li>
              ) : (
                sessions.map((session) => (
                  <li
                    key={session.id}
                    className={cn(
                      "grid grid-cols-[1fr_auto] items-start gap-3 py-4 px-4 hover:bg-accent/40 rounded-md transition-colors",
                      session.current && "bg-success/5 ring-1 ring-success/20"
                    )}
                  >
                    <div className="min-w-0 flex items-start gap-3">
                      {session.amr === "device_link" ? (
                        <Terminal className="mt-0.5 h-4 w-4 text-muted-foreground shrink-0" />
                      ) : (
                        <Monitor className="mt-0.5 h-4 w-4 text-muted-foreground shrink-0" />
                      )}
                      <div className="min-w-0 space-y-1">
                        <div className="flex items-center gap-2">
                          <div className="font-medium truncate">{session.browser}</div>
                          {session.current && (
                            <span className="rounded-full bg-success/15 text-success px-2.5 py-0.5 text-[11px] font-medium">
                              Current
                            </span>
                          )}
                        </div>
                        <div className="mt-1 grid grid-cols-[auto,1fr] gap-x-3 gap-y-1 text-sm text-muted-foreground">
                          {session.lastActivity && (
                            <>
                              <div className="text-muted-foreground/80">Last activity</div>
                              <div className="truncate pl-1.5">{session.lastActivity}</div>
                            </>
                          )}
                          {session.ipAddress && (
                            <>
                              <div className="text-muted-foreground/80">IP</div>
                              <div className="truncate">
                                {(() => {
                                  const copied = copiedSessionId === session.id;
                                  return (
                                    <button
                                      type="button"
                                      onClick={async () => {
                                        await navigator.clipboard.writeText(session.ipAddress!);
                                        setCopiedSessionId(session.id);
                                        window.setTimeout(() => setCopiedSessionId(null), 1500);
                                      }}
                                      className={cn(
                                        "inline-flex items-center gap-1 rounded px-1.5 py-0.5 font-mono focus:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                                        copied
                                          ? "bg-success/15 text-success ring-1 ring-success/30"
                                          : "bg-muted/70 hover:bg-muted"
                                      )}
                                      aria-live="polite"
                                      aria-label={
                                        copied ? "Copied" : `Copy IP ${session.ipAddress}`
                                      }
                                      title={copied ? "Copied!" : "Copy IP"}
                                    >
                                      {copied ? (
                                        <>
                                          <Check className="h-3 w-3" /> Copied
                                        </>
                                      ) : (
                                        session.ipAddress
                                      )}
                                    </button>
                                  );
                                })()}
                              </div>
                            </>
                          )}
                          {session.userAgent && (
                            <>
                              <div className="text-muted-foreground/80">User agent</div>
                              <div className="truncate pl-1.5">
                                <TooltipProvider>
                                  <Tooltip>
                                    <TooltipTrigger className="underline decoration-dotted">
                                      View
                                    </TooltipTrigger>
                                    <TooltipContent className="max-w-xs break-words">
                                      {session.userAgent}
                                    </TooltipContent>
                                  </Tooltip>
                                </TooltipProvider>
                              </div>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                    {!session.current && (
                      <div className="self-center">
                        <Button
                          size="sm"
                          variant="ghost"
                          className="min-w-[7.5rem] justify-center"
                          onClick={() => handleEndSession(session.id)}
                          aria-label={`End session on ${session.browser}`}
                        >
                          End session
                        </Button>
                      </div>
                    )}
                  </li>
                ))
              )}
            </ul>
          </section>
        </div>

        {/* Recovery Keys */}
        <section className="mt-6 space-y-4 pt-6 border-t border-border/40">
          <div className="flex items-baseline justify-between flex-wrap gap-2">
            <h2 className="text-xl font-bold">Recovery Keys</h2>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="destructive" size="sm">
                  <RotateCcw className="mr-2 h-4 w-4" />
                  Regenerate
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Regenerate recovery keys?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will replace your existing recovery keys. Old keys will no longer work.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handleRegenerateKeys}
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    Regenerate
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
          <div className="flex items-center gap-2">
            <p className="text-sm text-muted-foreground/70">
              {recoveryKeysGenerated
                ? `Last regenerated ${format(lastRecoveryDate, "PP")}`
                : "Never generated"}
            </p>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger>
                  <Info className="h-4 w-4 text-muted-foreground" />
                </TooltipTrigger>
                <TooltipContent>
                  <p>
                    Regenerating creates new keys and invalidates old ones.
                    <br />
                    New keys are shown only once.
                  </p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
          <div className="flex items-center gap-2 p-3 rounded-lg bg-info/10 border border-info/30">
            <Info className="h-4 w-4 text-info shrink-0" />
            <p className="text-sm text-foreground">
              Your recovery keys can only be viewed right after they're generated. If you've lost
              them, regenerate to create new keys.
            </p>
          </div>
        </section>

        {/* Danger Zone */}
        <section className="mt-8 space-y-4">
          <h2 className="text-xl font-bold text-destructive">Danger Zone</h2>
          <p className="text-sm text-muted-foreground">
            Once you delete your account, there is no going back.
          </p>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="destructive"
                className="w-full sm:w-auto"
                aria-describedby="delete-account-description"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete account
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
                <AlertDialogDescription id="delete-account-description">
                  This action cannot be undone. This will permanently delete your account and remove
                  all associated data from our servers.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleDeleteAccount}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  Yes, delete account
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </section>
      </div>

      <ProfileEditDialog
        open={isEditDialogOpen}
        onOpenChange={setIsEditDialogOpen}
        currentName={fullName}
        onSave={handleSave}
      />

      {/* Rename device dialog */}
      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle>Rename device</DialogTitle>
            <DialogDescription id="rename-device-description">
              Update the display name for this passkey.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 pt-2">
            <Input
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              placeholder="Device name"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  void submitRename();
                }
              }}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRenameOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => void submitRename()}
              disabled={
                renameValue.trim().length === 0 ||
                (renameTarget && renameValue.trim() === renameTarget.device_name)
              }
            >
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Profile;
