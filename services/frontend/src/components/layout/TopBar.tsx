import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";
import { CommandPalette } from "./CommandPalette";
import { ProfileDropdown } from "./ProfileDropdown";
import { useState } from "react";

export const TopBar = () => {
  const [open, setOpen] = useState(false);

  return (
    <div className="flex items-center gap-2 w-full">
      <div className="relative max-w-xl w-full">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          placeholder="Search or jump to… (⌘K)"
          onFocus={() => setOpen(true)}
          className="w-full rounded-full shadow-sm pl-9 focus:ring-2 focus:ring-ring focus:ring-offset-2"
          aria-label="Omnibar"
        />
      </div>
      <div className="ml-auto">
        <ProfileDropdown />
      </div>
      <CommandPalette open={open} onOpenChange={setOpen} />
    </div>
  );
};
