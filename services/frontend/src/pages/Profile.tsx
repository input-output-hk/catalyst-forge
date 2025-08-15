import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
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
import { Key, Monitor, RotateCcw, Trash2, MoreVertical, Edit, LogOut, Info } from "lucide-react";
import { format } from "date-fns";
import { generateRecoveryKeys, keysToText } from "@/lib/recovery";

// Mock session data
const mockSessions = [
  {
    id: "sess_1",
    browser: "Chrome 120.0.6099.234",
    os: "macOS 14.2.1",
    location: "San Francisco, CA",
    lastActivity: "2 minutes ago",
    loginDate: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
    current: true,
  },
  {
    id: "sess_2", 
    browser: "Safari 17.2",
    os: "iOS 17.2.1",
    location: "San Francisco, CA",
    lastActivity: "1 hour ago",
    loginDate: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(),
    current: false,
  },
];

const Profile = () => {
  const { state, actions } = useAppStore();
  const { toast } = useToast();
  const helmet = usePageTitle("My Profile – Catalyst Forge", "Manage your account information, devices, and security.", "/profile");

  const [fullName, setFullName] = useState(
    state.session.user ? state.session.user.split("@")[0].replace(".", " ") : ""
  );
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [sessions, setSessions] = useState(mockSessions);
  const [recoveryKeysGenerated, setRecoveryKeysGenerated] = useState(true);
  const [lastRecoveryDate] = useState(new Date(Date.now() - 1000 * 60 * 60 * 24 * 30));

  const userInitials = state.session.user
    ? state.session.user
        .split("@")[0]
        .split(".")
        .map((part) => part[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "U";

  const handleSave = async (newName: string) => {
    setFullName(newName);
    toast({ title: "Profile updated successfully" });
  };

  const handleLogoutAllDevices = () => {
    toast({ title: "Logged out of all devices" });
  };

  const handleRegenerateKeys = () => {
    const newKeys = generateRecoveryKeys(8);
    actions.startRecoveryGate(newKeys);
    setRecoveryKeysGenerated(true);
    toast({ title: "Recovery keys regenerated", description: "Make sure to save them securely." });
  };


  const handleEndSession = (sessionId: string) => {
    setSessions(prev => prev.filter(s => s.id !== sessionId));
    toast({ title: "Session terminated" });
  };

  const handleTerminateAllSessions = () => {
    setSessions(prev => prev.filter(s => s.current));
    toast({ title: "All other sessions terminated" });
  };

  const handleDeleteAccount = () => {
    toast({ title: "Account deletion initiated", description: "Your account will be deleted within 24 hours." });
  };

  return (
    <div className="container mx-auto py-6 max-w-6xl">
      {helmet}
      
      <div className="space-y-8">
        {/* Header */}
        <section className="flex items-center justify-between gap-4 pb-6 border-b border-border/40">
          <div className="flex items-center gap-4 min-w-0">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Avatar className="h-12 w-12 shrink-0 cursor-pointer hover:opacity-80 transition-opacity">
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
                <span className="rounded-full bg-foreground/10 px-2 py-0.5 text-xs">Administrator</span>
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
          {/* Authenticated Devices */}
          <section className="space-y-4">
            <div className="flex items-baseline justify-between flex-wrap gap-2">
              <h2 className="text-xl font-bold">Authenticated Devices</h2>
              <div className="flex gap-2">
                <Button 
                  size="sm"
                  onClick={() => {
                    actions.addDevice("New Device");
                    toast({ title: "Device added successfully", description: "The new device has been registered to your account." });
                  }}
                >
                  + Add device
                </Button>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button size="sm" variant="outline" className="text-destructive">
                      Log out of all devices
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Log out of all devices?</AlertDialogTitle>
                      <AlertDialogDescription>
                        This will sign you out of all devices and sessions. You'll need to sign in again on each device.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction onClick={handleLogoutAllDevices} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                        Log out all
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </div>
            <ul className="divide-y divide-border/40 rounded-lg border border-border/40">
              {state.devices.length === 0 ? (
                <li className="py-6 px-3 text-sm text-muted-foreground">No devices yet.</li>
              ) : (
                state.devices.map((device) => (
                  <li key={device.id} className="flex items-center justify-between py-3 px-3">
                    <div className="min-w-0 flex items-center gap-3">
                      <Key className="h-4 w-4 text-muted-foreground shrink-0" />
                      <div className="min-w-0">
                        <div className="font-medium">{device.label}</div>
                        <div className="text-xs text-muted-foreground">Added {format(new Date(device.addedAt), "PP")}</div>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button size="sm" variant="ghost" aria-label={`Rename ${device.label}`}>Rename</Button>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button size="sm" variant="ghost" className="text-destructive" aria-label={`Delete ${device.label}`}>Delete</Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Remove device?</AlertDialogTitle>
                            <AlertDialogDescription>
                              This will remove "{device.label}" from your account. You'll need to re-authenticate if you want to use this device again.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction 
                              onClick={() => actions.removeDevice(device.id)}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            >
                              Remove
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
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
                  <Button size="sm" variant="outline" className="whitespace-nowrap text-destructive">
                    Terminate all sessions
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Terminate all sessions?</AlertDialogTitle>
                    <AlertDialogDescription>
                      This will end all other active sessions except your current one. Other devices will need to sign in again.
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
                  <li key={session.id} className="grid grid-cols-[1fr_auto] items-center gap-3 py-3 px-3">
                    <div className="min-w-0 flex items-center gap-3">
                      <Monitor className="h-4 w-4 text-muted-foreground shrink-0" />
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <div className="font-medium truncate">{session.browser}</div>
                          {session.current && <span className="rounded-full bg-foreground/10 px-2 py-0.5 text-[10px]">Current</span>}
                        </div>
                        <div className="text-xs text-muted-foreground truncate">
                          {session.os} • {session.location} • Last activity {session.lastActivity}
                        </div>
                      </div>
                    </div>
                    {!session.current && (
                      <Button 
                        size="sm" 
                        variant="ghost" 
                        className="justify-self-end"
                        onClick={() => handleEndSession(session.id)}
                        aria-label={`End session on ${session.browser}`}
                      >
                        End session
                      </Button>
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
                : "Never generated"
              }
            </p>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger>
                  <Info className="h-4 w-4 text-muted-foreground" />
                </TooltipTrigger>
                <TooltipContent>
                  <p>Regenerating creates new keys and invalidates old ones.<br />New keys are shown only once.</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
          <div className="flex items-center gap-2 p-3 rounded-lg bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800">
            <Info className="h-4 w-4 text-blue-600 dark:text-blue-400 shrink-0" />
            <p className="text-sm text-blue-800 dark:text-blue-200">
              Your recovery keys can only be viewed right after they're generated. If you've lost them, regenerate to create new keys.
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
              <Button variant="destructive" className="w-full sm:w-auto" aria-describedby="delete-account-description">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete account
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
                <AlertDialogDescription id="delete-account-description">
                  This action cannot be undone. This will permanently delete your account and remove all associated data from our servers.
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
    </div>
  );
};

export default Profile;