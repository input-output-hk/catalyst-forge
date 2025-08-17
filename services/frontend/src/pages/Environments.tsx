import { useAppStore } from "@/store/app-store";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { usePageTitle } from "@/hooks/usePageTitle";

const Environments = () => {
  const { state } = useAppStore();
  const helmet = usePageTitle(
    "Environments – Catalyst Forge",
    "All deployment environments.",
    "/environments"
  );

  return (
    <section className="container py-8">
      {helmet}
      <h1 className="text-2xl font-bold mb-1">Environments</h1>
      <span className="forge-arc mt-1 mb-4 block" aria-hidden />
      <Card>
        <Table>
          <TableHeader className="sticky top-[56px] bg-background">
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Region</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {state.environments.map((e) => (
              <TableRow key={e.id}>
                <TableCell className="font-medium">{e.name}</TableCell>
                <TableCell>{e.region}</TableCell>
                <TableCell>
                  <Badge
                    variant={
                      e.status === "ready"
                        ? "secondary"
                        : e.status === "provisioning"
                          ? "outline"
                          : "destructive"
                    }
                  >
                    {e.status}
                  </Badge>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
    </section>
  );
};

export default Environments;
