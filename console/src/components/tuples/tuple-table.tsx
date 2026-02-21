"use client";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Trash2, Loader2 } from "lucide-react";
import { TupleDisplay } from "./tuple-display";
import { useDeleteTuple } from "@/lib/hooks/use-tuples";
import { toast } from "sonner";
import type { RelationTuple } from "@/lib/api/types";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

interface TupleTableProps {
  tenantId: string;
  tuples: RelationTuple[];
}

export function TupleTable({ tenantId, tuples }: TupleTableProps) {
  const deleteTuple = useDeleteTuple(tenantId);
  const [deleteTarget, setDeleteTarget] = useState<RelationTuple | null>(null);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteTuple.mutateAsync({
        object_type: deleteTarget.object_type,
        object_id: deleteTarget.object_id,
        relation: deleteTarget.relation,
        subject_type: deleteTarget.subject_type,
        subject_id: deleteTarget.subject_id,
        subject_relation: deleteTarget.subject_relation,
      });
      toast.success("Tuple deleted");
      setDeleteTarget(null);
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to delete tuple");
    }
  };

  if (tuples.length === 0) {
    return (
      <div className="text-center py-8 text-sm text-muted-foreground">
        No tuples found. Create one to get started.
      </div>
    );
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Tuple</TableHead>
            <TableHead className="w-[160px]">Created</TableHead>
            <TableHead className="w-[60px]" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {tuples.map((tuple, i) => (
            <TableRow key={i}>
              <TableCell>
                <TupleDisplay
                  objectType={tuple.object_type}
                  objectId={tuple.object_id}
                  relation={tuple.relation}
                  subjectType={tuple.subject_type}
                  subjectId={tuple.subject_id}
                  subjectRelation={tuple.subject_relation}
                />
              </TableCell>
              <TableCell className="text-xs text-muted-foreground">
                {new Date(tuple.created_at).toLocaleDateString()}
              </TableCell>
              <TableCell>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
                  onClick={() => setDeleteTarget(tuple)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Tuple</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Are you sure you want to delete this tuple? This action cannot be
            undone.
          </p>
          {deleteTarget && (
            <div className="py-2">
              <TupleDisplay
                objectType={deleteTarget.object_type}
                objectId={deleteTarget.object_id}
                relation={deleteTarget.relation}
                subjectType={deleteTarget.subject_type}
                subjectId={deleteTarget.subject_id}
                subjectRelation={deleteTarget.subject_relation}
              />
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteTuple.isPending}
            >
              {deleteTuple.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
