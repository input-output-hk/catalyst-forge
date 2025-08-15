import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { useAppStore } from "@/store/app-store";
import { useState } from "react";

export const FeatureFlagsPanel = () => {
  const { state, actions } = useAppStore();
  const [open, setOpen] = useState(false);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button className="fixed bottom-4 right-4 shadow-glow" variant="hero">Flags</Button>
      </SheetTrigger>
      <SheetContent side="right" className="w-96">
        <SheetHeader>
          <SheetTitle>Feature Flags</SheetTitle>
        </SheetHeader>
        <div className="mt-6 space-y-5">
          <div className="flex items-center justify-between">
            <Label htmlFor="dark">Dark mode</Label>
            <Switch id="dark" checked={state.flags.darkMode} onCheckedChange={() => actions.toggleFlag("darkMode")} />
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="compact">Compact density</Label>
            <Switch id="compact" checked={state.flags.compact} onCheckedChange={() => actions.toggleFlag("compact")} />
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="stream">Stream logs</Label>
            <Switch id="stream" checked={state.flags.streamLogs} onCheckedChange={() => actions.toggleFlag("streamLogs")} />
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
};
