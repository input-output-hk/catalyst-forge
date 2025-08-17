import { useEffect, useMemo, useRef, useState } from "react";
import { useAppStore } from "@/store/app-store";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { LogViewer } from "@/components/jobs/LogViewer";

const Jobs = () => {
  const { state } = useAppStore();
  const [selected, setSelected] = useState<string | null>(state.jobs[0]?.id ?? null);
  const job = useMemo(() => state.jobs.find((j) => j.id === selected), [state.jobs, selected]);
  const helmet = usePageTitle("Jobs – Catalyst Forge", "Run history and streaming logs.", "/jobs");

  return (
    <section className="container py-8">
      {helmet}
      <h1 className="text-2xl font-bold mb-1">Jobs</h1>
      <span className="forge-arc mt-1 mb-4 block" aria-hidden />
      <div className="grid lg:grid-cols-2 gap-4">
        <Card>
          <Table>
            <TableHeader className="sticky top-[56px] bg-background">
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Service</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Started</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {state.jobs.map((j) => (
                <TableRow
                  key={j.id}
                  className={selected === j.id ? "bg-muted" : ""}
                  onClick={() => setSelected(j.id)}
                >
                  <TableCell className="font-mono text-xs">{j.id}</TableCell>
                  <TableCell>{j.serviceId}</TableCell>
                  <TableCell>{j.status}</TableCell>
                  <TableCell>{new Date(j.startedAt).toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
        <Card className="p-4">
          {job ? (
            <LogViewer run={job} />
          ) : (
            <div className="text-muted-foreground">Select a run to view details</div>
          )}
        </Card>
      </div>
    </section>
  );
};

export default Jobs;
