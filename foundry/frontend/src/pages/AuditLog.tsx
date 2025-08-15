import { useAppStore } from "@/store/app-store";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { useMemo, useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";

const AuditLog = () => {
  const { state } = useAppStore();
  const [actor, setActor] = useState("");
  const [action, setAction] = useState("");
  const [resource, setResource] = useState("");
  const helmet = usePageTitle("Audit Log – Catalyst Forge", "Filterable timeline of actions.", "/audit");

  const rows = useMemo(() => state.audit.filter((e) =>
    (!actor || e.actor.includes(actor)) && (!action || e.action.includes(action)) && (!resource || e.resource.includes(resource))
  ), [state.audit, actor, action, resource]);

  return (
    <section className="container py-8">
      {helmet}
      <h1 className="text-2xl font-bold mb-1">Audit Log</h1>
      <span className="forge-arc mt-1 mb-4 block" aria-hidden />
      <div className="grid md:grid-cols-4 gap-2 mb-4">
        <Input placeholder="Actor" value={actor} onChange={(e) => setActor(e.target.value)} />
        <Input placeholder="Action" value={action} onChange={(e) => setAction(e.target.value)} />
        <Input placeholder="Resource" value={resource} onChange={(e) => setResource(e.target.value)} />
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Actor</TableHead>
            <TableHead>Action</TableHead>
            <TableHead>Resource</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((e) => (
            <TableRow key={e.id}>
              <TableCell>{new Date(e.timestamp).toLocaleString()}</TableCell>
              <TableCell>{e.actor}</TableCell>
              <TableCell>{e.action}</TableCell>
              <TableCell>{e.resource}</TableCell>
            </TableRow>
          ))}
          {rows.length === 0 && (
            <TableRow>
              <TableCell colSpan={4} className="text-muted-foreground">No results</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </section>
  );
};

export default AuditLog;
