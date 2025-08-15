import { useAppStore } from "@/store/app-store";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { toast } from "@/hooks/use-toast";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useState } from "react";

const Secrets = () => {
  const { state, actions } = useAppStore();
  const [revealValue, setRevealValue] = useState<string | null>(null);
  const [rotateId, setRotateId] = useState<string | null>(null);
  const helmet = usePageTitle("Secrets – Catalyst Forge", "Manage and rotate secrets.", "/secrets");

  return (
    <section className="container py-8">
      {helmet}
      <h1 className="text-2xl font-bold mb-1">Secrets</h1>
      <span className="forge-arc mt-1 mb-4 block" aria-hidden />
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Key</TableHead>
            <TableHead>Version</TableHead>
            <TableHead>Value</TableHead>
            <TableHead>Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {state.secrets.map((s) => (
            <TableRow key={s.id}>
              <TableCell className="font-mono">{s.key}</TableCell>
              <TableCell>v{s.version}</TableCell>
              <TableCell><span className="blur-sm select-all">{s.value}</span></TableCell>
              <TableCell className="space-x-2">
                <Button size="sm" variant="secondary" onClick={async () => setRevealValue(await actions.revealSecret(s.id))}>Reveal</Button>
                <Button size="sm" variant="outline" onClick={() => setRotateId(s.id)}>Rotate</Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={revealValue !== null} onOpenChange={() => setRevealValue(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revealed value</DialogTitle>
          </DialogHeader>
          <Input readOnly value={revealValue ?? ""} className="font-mono" />
        </DialogContent>
      </Dialog>

      <Dialog open={rotateId !== null} onOpenChange={() => setRotateId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rotate secret</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">This will create a new version and revoke the previous one.</p>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRotateId(null)}>Cancel</Button>
            <Button onClick={async () => { if (rotateId) { await actions.rotateSecret(rotateId); toast({ title: "Rotated", description: "A new version was created." }); setRotateId(null); } }}>Confirm</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
};

export default Secrets;
