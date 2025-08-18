import { useEffect, useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { rbacApi, type Role, type RoleDef, type ExplainRequest } from "@/lib/api/rbac";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useToast } from "@/components/ui/use-toast";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Pencil, Plus, Save } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";

export default function RBACAdmin() {
    return (
        <div className="p-4 space-y-6">
            <h1 className="text-2xl font-semibold tracking-tight">RBAC Administration</h1>
            <Tabs defaultValue="roles" className="space-y-6">
                <TabsList>
                    <TabsTrigger value="roles">Roles</TabsTrigger>
                    <TabsTrigger value="explain">Explain</TabsTrigger>
                </TabsList>
                <TabsContent value="roles">
                    <RolesPanel />
                </TabsContent>
                <TabsContent value="explain">
                    <ExplainPanel />
                </TabsContent>
            </Tabs>
        </div>
    );
}

function RolesPanel() {
    const qc = useQueryClient();
    const { toast } = useToast();
    const [editorOpen, setEditorOpen] = useState(false);
    const [editingSlug, setEditingSlug] = useState<string | null>(null);
    const [form, setForm] = useState<{ slug: string; name: string; description: string; color: string } | null>(null);
    const rolesQ = useQuery({
        queryKey: ["rbac", "roles"],
        queryFn: async () => {
            const res = await rbacApi.listRoles();
            if (!res.response.ok) throw new Error("Failed to load roles");
            return res.data?.roles ?? [];
        },
    });

    // Placeholder for future mutations (create/update role)

    return (
        <Card>
            <CardHeader className="px-4 pt-3 pb-2">
                <div className="flex items-center justify-between gap-2">
                    <CardTitle>Roles</CardTitle>
                    <Button
                        variant="hero"
                        size="sm"
                        aria-label="Add role"
                        onClick={() => {
                            setEditingSlug(null);
                            setForm({ slug: "", name: "", description: "", color: "#7c3aed" });
                            setEditorOpen(true);
                        }}
                    >
                        <Plus className="h-4 w-4 mr-2" /> Add Role
                    </Button>
                </div>
            </CardHeader>
            <CardContent className="px-4 pt-0 pb-3">
                <div className="overflow-x-auto">
                    <Table className="[&_td]:px-4 [&_td]:py-3 [&_th]:px-4 [&_th]:py-3">
                        <TableHeader className="sticky top-0 z-10">
                            <TableRow className="bg-gradient-to-b from-white/[0.16] to-white/[0.08] text-foreground/95 border-b border-white/25 shadow-[inset_0_-1px_0_0_rgba(255,255,255,0.2)]">
                                <TableHead className="w-40 uppercase tracking-wide text-[12px] text-foreground/90 font-semibold">Slug</TableHead>
                                <TableHead className="w-56 uppercase tracking-wide text-[12px] text-foreground/90 font-semibold">Name</TableHead>
                                <TableHead className="w-[55%] uppercase tracking-wide text-[12px] text-foreground/90 font-semibold">Description</TableHead>
                                <TableHead className="text-right w-24 uppercase tracking-wide text-[12px] text-foreground/90 font-semibold">Version</TableHead>
                                <TableHead className="text-right w-20 uppercase tracking-wide text-[12px] text-foreground/90 font-semibold">Actions</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {rolesQ.data?.map((r) => (
                                <TableRow
                                    key={r.slug}
                                    className="h-10 even:bg-white/[0.04] hover:bg-white/[0.06] transition-colors border-b border-white/10 last:border-0"
                                >
                                    <TableCell className="font-semibold whitespace-nowrap">
                                        {r.color ? (
                                            <span
                                                className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold"
                                                style={{
                                                    backgroundColor: r.color,
                                                    color: '#ffffff',
                                                    border: '1px solid rgba(255,255,255,0.25)',
                                                    textShadow: '0 1px 0 rgba(0,0,0,0.25)',
                                                }}
                                            >
                                                {r.slug}
                                            </span>
                                        ) : (
                                            r.slug
                                        )}
                                    </TableCell>
                                    <TableCell className="whitespace-nowrap">{r.name}</TableCell>
                                    <TableCell className="text-muted-foreground pr-4">{r.description}</TableCell>
                                    <TableCell className="text-right">
                                        <span className="inline-flex items-center rounded-md bg-white/5 px-2 py-0.5 text-xs font-medium tabular-nums">
                                            {r.version}
                                        </span>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-1.5">
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        aria-label={`Edit ${r.slug}`}
                                                        className="rounded-md text-foreground/90 hover:bg-white/10"
                                                        onClick={() => {
                                                            setEditingSlug(r.slug || null);
                                                            setForm({
                                                                slug: r.slug || "",
                                                                name: r.name || "",
                                                                description: r.description || "",
                                                                color: r.color || "#7c3aed",
                                                            });
                                                            setEditorOpen(true);
                                                        }}
                                                    >
                                                        <Pencil className="h-4 w-4" />
                                                    </Button>
                                                </TooltipTrigger>
                                                <TooltipContent>Edit role (coming soon)</TooltipContent>
                                            </Tooltip>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))}
                            {rolesQ.data?.length === 0 && (
                                <TableRow>
                                    <TableCell colSpan={5} className="text-center text-muted-foreground">
                                        No roles found
                                    </TableCell>
                                </TableRow>
                            )}
                        </TableBody>
                    </Table>
                </div>
            </CardContent>
            <RoleEditor
                open={editorOpen}
                onOpenChange={setEditorOpen}
                initial={form}
                onSaved={() => {
                    /* no-op, invalidation handled in editor */
                }}
            />
        </Card>
    );
}

function ExplainPanel() {
    const { toast } = useToast();
    const [subjectType, setSubjectType] = useState("user");
    const [subjectID, setSubjectID] = useState("");
    const [permission, setPermission] = useState("");
    const [resourceType, setResourceType] = useState("project");
    const [resourceID, setResourceID] = useState("");
    const [result, setResult] = useState<{ decision: string; steps: number } | null>(null);

    const explain = useMutation({
        mutationFn: async () => {
            const body: ExplainRequest = {
                subject: { type: subjectType, id: subjectID },
                permission,
                resource: { type: resourceType, id: resourceID },
            };
            const res = await rbacApi.explain(body);
            if (res.response.status === 428) {
                throw new Error("Step-up required (428)");
            }
            if (!res.response.ok || !res.data) throw new Error("Explain failed");
            return res.data;
        },
        onSuccess: (data: import("@/lib/api/rbac").ExplainResponse | undefined) => {
            const traceUnknown = data?.trace as { steps?: unknown } | undefined;
            const stepsValue = traceUnknown?.steps;
            const steps = Array.isArray(stepsValue) ? stepsValue.length : 0;
            setResult({ decision: String(data?.decision ?? ""), steps });
        },
        onError: (e: unknown) => {
            toast({ description: e instanceof Error ? e.message : "Explain failed" });
            setResult(null);
        },
    });

    return (
        <Card>
            <CardHeader>
                <CardTitle>Explain Decision</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                        <label className="text-sm font-medium">Subject Type</label>
                        <Input value={subjectType} onChange={(e) => setSubjectType(e.target.value)} placeholder="user|group|service" />
                        <label className="text-sm font-medium">Subject ID</label>
                        <Input value={subjectID} onChange={(e) => setSubjectID(e.target.value)} placeholder="uuid-or-id" />
                        <label className="text-sm font-medium">Permission</label>
                        <Input value={permission} onChange={(e) => setPermission(e.target.value)} placeholder="e.g. infra:deploy" />
                    </div>
                    <div className="space-y-2">
                        <label className="text-sm font-medium">Resource Type</label>
                        <Input value={resourceType} onChange={(e) => setResourceType(e.target.value)} placeholder="project|org|resource|any" />
                        <label className="text-sm font-medium">Resource ID</label>
                        <Input value={resourceID} onChange={(e) => setResourceID(e.target.value)} placeholder="id (optional)" />
                    </div>
                </div>
                <Separator className="my-4" />
                <Button onClick={() => explain.mutate()} disabled={explain.isPending}>
                    {explain.isPending ? "Explaining…" : "Explain"}
                </Button>
                {result && (
                    <div className="mt-4 text-sm text-muted-foreground">
                        Decision: <span className="font-medium text-foreground">{result.decision}</span>, Steps: {result.steps}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}

function RoleEditor({
    open,
    onOpenChange,
    initial,
    onSaved,
}: {
    open: boolean;
    onOpenChange: (o: boolean) => void;
    initial: { slug: string; name: string; description: string; color: string } | null;
    onSaved: () => void;
}) {
    const { toast } = useToast();
    const qc = useQueryClient();
    const [values, setValues] = useState(initial ?? { slug: "", name: "", description: "", color: "#7c3aed" });
    const permsQ = useQuery({
        queryKey: ["rbac", "permissions"],
        queryFn: async () => {
            const res = await rbacApi.listPermissions();
            if (!res.response.ok) throw new Error("Failed to load permissions");
            return res.data?.permissions ?? [];
        },
    });
    const isEdit = Boolean(initial && initial.slug);

    // Keep form in sync when opening with new initial
    useEffect(() => {
        if (open) setValues(initial ?? { slug: "", name: "", description: "", color: "#7c3aed" });
    }, [open, initial]);

    const save = useMutation({
        mutationFn: async () => {
            const payload: import("@/lib/api/rbac").RoleDef = {
                slug: values.slug,
                name: values.name,
                description: values.description,
                color: values.color,
                version: 1,
                entries: [],
            };
            if (isEdit) {
                const res = await rbacApi.updateRole(values.slug, payload);
                if (!res.response.ok) throw new Error("Update failed");
            } else {
                const res = await rbacApi.createRole(payload);
                if (res.response.status !== 201) throw new Error("Create failed");
            }
        },
        onSuccess: async () => {
            await qc.invalidateQueries({ queryKey: ["rbac", "roles"] });
            toast({ description: isEdit ? "Role updated" : "Role created" });
            onOpenChange(false);
            onSaved();
        },
        onError: (e: unknown) => {
            toast({ description: e instanceof Error ? e.message : "Save failed", variant: "destructive" });
        },
    });

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>{isEdit ? "Edit Role" : "Add Role"}</DialogTitle>
                </DialogHeader>
                <div className="space-y-3">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                        <div>
                            <Label htmlFor="slug">Slug</Label>
                            <Input
                                id="slug"
                                value={values.slug}
                                onChange={(e) => setValues((v) => ({ ...v, slug: e.target.value }))}
                                placeholder="e.g. developer"
                                disabled={isEdit}
                            />
                        </div>
                        <div>
                            <Label htmlFor="name">Name</Label>
                            <Input
                                id="name"
                                value={values.name}
                                onChange={(e) => setValues((v) => ({ ...v, name: e.target.value }))}
                                placeholder="Developer"
                            />
                        </div>
                    </div>
                    <div>
                        <Label htmlFor="description">Description</Label>
                        <Input
                            id="description"
                            value={values.description}
                            onChange={(e) => setValues((v) => ({ ...v, description: e.target.value }))}
                            placeholder="Short description"
                        />
                    </div>
                    <div className="grid grid-cols-[1fr_auto] gap-3 items-end">
                        <div>
                            <Label htmlFor="color">Color</Label>
                            <Input
                                id="color"
                                value={values.color}
                                onChange={(e) => setValues((v) => ({ ...v, color: e.target.value }))}
                                placeholder="#7c3aed"
                            />
                        </div>
                        <div className="flex items-center justify-end">
                            <span
                                className="inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold"
                                style={{ backgroundColor: values.color, color: '#fff', border: '1px solid rgba(255,255,255,0.25)' }}
                            >
                                {values.slug || "preview"}
                            </span>
                        </div>
                    </div>
                    <div>
                        <Label>Entries</Label>
                        <div className="mt-2 space-y-2">
                            <div className="rounded-md border border-white/10 p-3">
                                <div className="grid grid-cols-1 sm:grid-cols-4 gap-2 items-center">
                                    <div>
                                        <Label className="text-xs">Effect</Label>
                                        <Select defaultValue="allow" disabled>
                                            <SelectTrigger>
                                                <SelectValue placeholder="allow" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="allow">allow</SelectItem>
                                                <SelectItem value="deny">deny</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <div className="sm:col-span-2">
                                        <Label className="text-xs">Permission</Label>
                                        <Select disabled={!permsQ.data || permsQ.data.length === 0}>
                                            <SelectTrigger>
                                                <SelectValue placeholder={permsQ.isLoading ? "Loading…" : "Select permission"} />
                                            </SelectTrigger>
                                            <SelectContent className="max-h-64">
                                                {(permsQ.data ?? []).map((p) => (
                                                    <SelectItem key={p} value={p}>
                                                        {p}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <div>
                                        <Label className="text-xs">Resource Type</Label>
                                        <Input placeholder="optional" />
                                    </div>
                                </div>
                                <div className="mt-2 text-xs text-muted-foreground">
                                    Conditions editor coming next.
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <DialogFooter className="mt-4">
                    <Button variant="secondary" onClick={() => onOpenChange(false)}>
                        Cancel
                    </Button>
                    <Button onClick={() => save.mutate()} disabled={save.isPending || !values.slug || !values.name}>
                        <Save className="h-4 w-4 mr-2" /> {isEdit ? "Save" : "Create"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

