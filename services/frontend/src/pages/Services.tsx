import { useAppStore } from "@/store/app-store";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { useNavigate } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";

const Status = ({ s }: { s: "healthy" | "degraded" | "down" }) => (
  <Badge variant={s === "healthy" ? "secondary" : s === "degraded" ? "outline" : "destructive"}>{s}</Badge>
);

const Services = () => {
  const { state } = useAppStore();
  const nav = useNavigate();
  const helmet = usePageTitle("Services – Catalyst Forge", "Browse services and quick details.", "/services");

  return (
    <section className="container py-8">
      {helmet}
      <h1 className="text-2xl font-bold mb-1">Services</h1>
      <span className="forge-arc mt-1 mb-4 block" aria-hidden />
      <Card>
        <Table>
          <TableHeader className="sticky top-[56px] bg-background">
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Owner</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Envs</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {state.services.map((s) => (
              <TableRow key={s.id} className="cursor-pointer" onClick={() => nav(`/jobs?service=${s.id}`)}>
                <TableCell className="font-medium">{s.name}</TableCell>
                <TableCell>{s.owner}</TableCell>
                <TableCell><Status s={s.status} /></TableCell>
                <TableCell>{s.envs.join(", ")}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
    </section>
  );
};

export default Services;
