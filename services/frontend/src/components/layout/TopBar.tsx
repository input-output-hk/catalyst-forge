import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CommandPalette } from "./CommandPalette";
import { ProfileDropdown } from "./ProfileDropdown";
import { useState } from "react";

export const TopBar = () => {
  const [open, setOpen] = useState(false);

  return (
    <div className="flex items-center gap-2 w-full">
      <div className="relative max-w-xl w-full">
        <Input
          placeholder="Search or jump to… (⌘K)"
          onFocus={() => setOpen(true)}
          className="w-full rounded-full shadow-sm"
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
