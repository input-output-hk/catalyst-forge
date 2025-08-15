import * as React from "react";
import { CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { useNavigate } from "react-router-dom";
import { LogoMark } from "@/components/brand/LogoMark";

const entries = [
  { label: "Welcome", to: "/welcome" },
  { label: "Dashboard", to: "/" },
  { label: "Services", to: "/services" },
  { label: "Environments", to: "/environments" },
  { label: "Jobs", to: "/jobs" },
  { label: "Secrets", to: "/secrets" },
  { label: "Audit Log", to: "/audit" },
  { label: "Settings", to: "/settings" },
  { label: "Auth Flows", to: "/auth-demo" },
];

export function CommandPalette({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const navigate = useNavigate();

  React.useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        onOpenChange(true);
      }
      if (e.key === "Escape") onOpenChange(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onOpenChange]);

  return (
    <CommandDialog open={open} onOpenChange={onOpenChange}>
      <div className="flex items-center gap-3 px-3 pt-3 pb-2">
        <LogoMark size={24} />
        <div className="leading-tight">
          <div className="text-sm font-semibold">Catalyst Forge</div>
          <div className="text-xs text-muted-foreground">⌘K to jump</div>
        </div>
      </div>
      <CommandInput placeholder="Type a command or search…" />
      <CommandList>
        <CommandEmpty>No results.</CommandEmpty>
        <CommandGroup heading="Navigate">
          {entries.map((e) => (
            <CommandItem key={e.to} onSelect={() => { onOpenChange(false); navigate(e.to); }}>
              {e.label}
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
