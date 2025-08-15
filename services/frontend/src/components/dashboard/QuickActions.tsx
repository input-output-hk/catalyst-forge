import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { can } from "@/lib/mock/dashboard";
import { PlusSquare, Rocket, Shield, KeyRound } from "lucide-react";

function Gate({ action, children }: { action: Parameters<typeof can>[0]; children: React.ReactElement }) {
  const perm = can(action);
  if (perm.allowed) return children;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        {children}
      </TooltipTrigger>
      <TooltipContent>{perm.reason}</TooltipContent>
    </Tooltip>
  );
}

export default function QuickActions() {
  return (
    <div className="flex flex-wrap gap-2">
      <Gate action="create:preview">
        <Button size="sm" variant="secondary" disabled={!can("create:preview").allowed} aria-label="New Preview Env">
          <PlusSquare className="mr-2" /> New Preview Env
        </Button>
      </Gate>
      <Gate action="release:create">
        <Button size="sm" disabled={!can("release:create").allowed} aria-label="Create Release">
          <Rocket className="mr-2" /> Create Release
        </Button>
      </Gate>
      <Gate action="cert:request">
        <Button size="sm" variant="outline" disabled={!can("cert:request").allowed} aria-label="Request Cert">
          <Shield className="mr-2" /> Request Cert
        </Button>
      </Gate>
      <Gate action="secret:rotate">
        <Button size="sm" variant="outline" disabled={!can("secret:rotate").allowed} aria-label="Rotate Secret">
          <KeyRound className="mr-2" /> Rotate Secret
        </Button>
      </Gate>
    </div>
  );
}
