import { useAppStore } from "@/store/app-store";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";

const SettingsProfile = () => {
  const { state, actions } = useAppStore();
  const [label, setLabel] = useState("");
  const helmet = usePageTitle("Settings – Catalyst Forge", "Devices and preferences.", "/settings");

  return (
    <section className="container py-8">
      {helmet}
      <h1 className="text-2xl font-bold mb-1">Settings</h1>
      <span className="forge-arc mt-1 mb-4 block" aria-hidden />
      <div className="grid md:grid-cols-2 gap-6">
        <div>
          <h2 className="font-semibold mb-2">Registered devices</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Label</TableHead>
                <TableHead>Added</TableHead>
                <TableHead></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {state.devices.map((d) => (
                <TableRow key={d.id}>
                  <TableCell>{d.label}</TableCell>
                  <TableCell>{new Date(d.addedAt).toLocaleDateString()}</TableCell>
                  <TableCell className="text-right">
                    <Button size="sm" variant="outline" onClick={() => actions.removeDevice(d.id)}>Remove</Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="mt-4 flex gap-2">
            <Input placeholder="New credential label" value={label} onChange={(e) => setLabel(e.target.value)} />
            <Button onClick={() => label && actions.addDevice(label)}>Add credential</Button>
          </div>
        </div>
        <div>
          <h2 className="font-semibold mb-2">Preferences</h2>
          <p className="text-muted-foreground">Configure theme, density and more via the Flags panel at the bottom-right.</p>
        </div>
      </div>
    </section>
  );
};

export default SettingsProfile;
