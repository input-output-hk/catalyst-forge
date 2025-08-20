import { useMemo, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { useAppStore } from "@/store/app-store";
// recovery helpers removed
import { useIsMobile } from "@/hooks/use-mobile";
import { Copy, Download, Printer, ShieldAlert } from "lucide-react";
import { useNavigate } from "react-router-dom";

export const RecoveryKeysGate = () => {
  const { state, actions } = useAppStore();
  const { toast } = useToast();
  const [interacted, setInteracted] = useState(false);
  const navigate = useNavigate();
  const isMobile = useIsMobile();

  const keys: string[] = [];

  const printable = useMemo(() => keys.join("\n"), [keys]);

  const onCopyAll = async () => {
    try {
      await navigator.clipboard.writeText(printable);
      setInteracted(true);
      toast({ title: "Copied", description: "All recovery keys copied to clipboard." });
    } catch {
      toast({
        title: "Copy failed",
        description: "Could not copy to clipboard.",
        variant: "destructive",
      });
    }
  };

  const onDownload = () => {
    const blob = new Blob([printable], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "recovery-keys.txt";
    a.click();
    setInteracted(true);
    URL.revokeObjectURL(url);
  };

  const onPrint = () => {
    const w = window.open("", "_blank", "noopener,noreferrer");
    if (!w) return;
    w.document.write(
      `<pre style="font: 14px/1.4 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace; white-space: pre-wrap;">${printable.replace(/</g, "&lt;")}</pre>`
    );
    w.document.close();
    w.focus();
    setInteracted(true);
    w.print();
  };

  const canContinue = interacted; // gate: must perform one save action

  const onContinue = () => {
    if (!canContinue) return;
    actions.completeRecoveryGate();
    setInteracted(false);
    toast({
      title: "Device registered successfully",
      description: "Recovery keys saved. You're all set.",
    });
    const dest = state.recoveryGate.returnTo || window.location.pathname;
    navigate(dest);
  };

  return (
    <Dialog open={false} onOpenChange={() => { }}>
      <DialogContent
        className="sm:max-w-3xl max-w-[calc(100vw-2rem)] overflow-x-hidden sm:max-h-[80vh] sm:overflow-y-auto [&>button.absolute.right-4.top-4]:hidden"
        aria-describedby="recovery-description"
        onEscapeKeyDown={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldAlert className="size-5 text-warning" aria-hidden />
            Save these recovery keys — you will not see them again
          </DialogTitle>
          <DialogDescription id="recovery-description">
            If you lose access to your device, these keys are the only way to sign in. Treat them
            like passwords. Store them somewhere safe.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 pb-40 sm:pb-6">
          <section>
            <div className="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-4 gap-3">
              {keys.map((k, i) => (
                <div
                  key={i}
                  className="rounded-md border border-input bg-muted/60 shadow-sm px-3 py-2 min-h-16 min-w-0"
                >
                  <div className="text-xs text-muted-foreground">Key {i + 1}</div>
                  <code className="block font-mono text-xs sm:text-sm tracking-normal sm:tracking-wider text-foreground select-text leading-6 break-all">
                    {k}
                  </code>
                </div>
              ))}
            </div>
          </section>

          <section className="hidden sm:flex items-start justify-between gap-4">
            <div className="flex flex-col gap-2">
              <Button
                variant="secondary"
                onClick={onCopyAll}
                aria-label="Copy all recovery keys"
                className="shrink-0 border border-input hover:shadow-sm active:translate-y-px active:scale-95 transition-all"
              >
                <Copy /> {isMobile ? "Copy all" : "Copy all keys"}
              </Button>
              <Button
                variant="secondary"
                onClick={onDownload}
                aria-label="Download recovery keys as text file"
                className="shrink-0 border border-input hover:shadow-sm active:translate-y-px active:scale-95 transition-all"
              >
                <Download /> {isMobile ? "Download" : "Download keys"}
              </Button>
              <Button
                variant="secondary"
                onClick={onPrint}
                aria-label="Print recovery keys"
                className="shrink-0 border border-input hover:shadow-sm active:translate-y-px active:scale-95 transition-all"
              >
                <Printer /> {isMobile ? "Print" : "Print keys"}
              </Button>
            </div>
            <Button
              onClick={onContinue}
              disabled={!canContinue}
              size="lg"
              className={canContinue ? "animate-in fade-in-0" : ""}
            >
              Continue
            </Button>
          </section>

          <div
            className="sm:hidden sticky bottom-0 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-t pt-3 px-4 sm:px-6"
            style={{ paddingBottom: "max(env(safe-area-inset-bottom), 0.75rem)" }}
          >
            {/* Mobile actions in sticky bar */}
            <div className="mb-3 flex gap-2 justify-center sm:hidden flex-wrap">
              <Button
                variant="secondary"
                onClick={onCopyAll}
                aria-label="Copy all recovery keys"
                className="border border-input hover:shadow-sm active:translate-y-px active:scale-95 transition-all"
              >
                <Copy /> Copy all
              </Button>
              <Button
                variant="secondary"
                onClick={onDownload}
                aria-label="Download recovery keys as text file"
                className="border border-input hover:shadow-sm active:translate-y-px active:scale-95 transition-all"
              >
                <Download /> Download
              </Button>
              <Button
                variant="secondary"
                onClick={onPrint}
                aria-label="Print recovery keys"
                className="border border-input hover:shadow-sm active:translate-y-px active:scale-95 transition-all"
              >
                <Printer /> Print
              </Button>
            </div>
            <div className="flex justify-center sm:justify-end">
              <Button
                onClick={onContinue}
                disabled={!canContinue}
                size="lg"
                className={canContinue ? "animate-in fade-in-0" : ""}
              >
                Continue
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};
