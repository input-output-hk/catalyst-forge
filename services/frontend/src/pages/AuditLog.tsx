import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { useEffect, useMemo, useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";
import { forge } from "@/lib/client";
import type { paths } from "forge-client";

const AuditLog = () => {
  const [rows, setRows] = useState<Array<{ id: string; type: string; actor_id?: string; user_id?: string; metadata?: Record<string, unknown>; created_at: string }>>([]);
  const [loading, setLoading] = useState(false);
  const [actor, setActor] = useState("");
  const [types, setTypes] = useState("");
  const helmet = usePageTitle("Audit Log – Catalyst Forge", "Filterable timeline of actions.", "/audit");

  const filtered = useMemo(() => rows.filter((e) =>
    (!actor || (e.actor_id || "").includes(actor)) && (!types || types.split(",").some(t => e.type.includes(t.trim())))
  ), [rows, actor, types]);

  useEffect(() => {
    (async () => {
      setLoading(true);
      try {
        const res = await forge.raw.GET('/api/v1/admin/audit', {
          params: {
            query: {
              actor_id: actor.trim() || undefined,
              types: types.trim() || undefined,
              limit: 200,
            },
          },
        });
        if (!res.response.ok) return;
        type AuditResponse = paths["/api/v1/admin/audit"]["get"]["responses"][200]["content"]["application/json"];
        const data = res.data as AuditResponse | undefined;
        setRows((data?.events || []).map(e => ({
          id: e.id || "",
          type: e.type || "",
          actor_id: e.actor_id,
          user_id: e.user_id,
          metadata: (e.metadata as Record<string, unknown>) || undefined,
          created_at: e.created_at || "",
        })));
      } finally {
        setLoading(false);
      }
    })();
  }, [actor, types]);

  return (
    <section className="container py-8">
      {helmet}
      <h1 className="text-2xl font-bold mb-1">Audit Log</h1>
      <span className="forge-arc mt-1 mb-4 block" aria-hidden />
      <div className="grid md:grid-cols-3 gap-2 mb-4">
        <Input placeholder="Actor ID" value={actor} onChange={(e) => setActor(e.target.value)} />
        <Input placeholder="Types (comma-separated)" value={types} onChange={(e) => setTypes(e.target.value)} />
        <Input readOnly value={loading ? "Loading…" : `${filtered.length} events`} />
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Actor</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Meta</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((e) => (
            <TableRow key={e.id}>
              <TableCell>{new Date(e.created_at).toLocaleString()}</TableCell>
              <TableCell>{e.actor_id || ""}</TableCell>
              <TableCell>{e.type}</TableCell>
              <TableCell className="max-w-[420px] truncate">{e.metadata ? JSON.stringify(e.metadata) : ""}</TableCell>
            </TableRow>
          ))}
          {filtered.length === 0 && (
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
