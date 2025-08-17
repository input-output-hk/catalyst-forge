import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type Props = {
  selectedCount: number;
  onDisable: () => void;
  onEnable: () => void;
  onLogout: () => void;
  onChangeRole: (role: "admin" | "member") => void;
  onDelete: () => void;
};

export default function BulkActionsBar({
  selectedCount,
  onDisable,
  onEnable,
  onLogout,
  onChangeRole,
  onDelete,
}: Props) {
  const [open, setOpen] = useState(false);
  return (
    <div className="mt-3 flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2 text-sm">
      <div>
        <span className="font-medium">{selectedCount} selected</span>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="secondary" onClick={onDisable}>
          Disable
        </Button>
        <Button size="sm" variant="secondary" onClick={onEnable}>
          Re-enable
        </Button>
        <Button size="sm" variant="secondary" onClick={onLogout}>
          Force log out
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" variant="outline">
              Change role
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => onChangeRole("admin")}>Admin</DropdownMenuItem>
            <DropdownMenuItem onClick={() => onChangeRole("member")}>Member</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button size="sm" variant="destructive" onClick={() => setOpen(true)}>
          Delete
        </Button>
        <AlertDialog open={open}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete</AlertDialogTitle>
              <AlertDialogDescription>
                This removes selected users and their credentials.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={() => setOpen(false)}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={() => {
                  onDelete();
                  setOpen(false);
                }}
              >
                Confirm
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
}
