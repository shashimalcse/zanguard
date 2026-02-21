"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Plus, Loader2 } from "lucide-react";
import { useWriteTuple } from "@/lib/hooks/use-tuples";
import { toast } from "sonner";

interface CreateTupleDialogProps {
  tenantId: string;
}

export function CreateTupleDialog({ tenantId }: CreateTupleDialogProps) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    object_type: "",
    object_id: "",
    relation: "",
    subject_type: "",
    subject_id: "",
    subject_relation: "",
  });
  const writeTuple = useWriteTuple(tenantId);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await writeTuple.mutateAsync({
        object_type: form.object_type,
        object_id: form.object_id,
        relation: form.relation,
        subject_type: form.subject_type,
        subject_id: form.subject_id,
        subject_relation: form.subject_relation || undefined,
      });
      toast.success("Tuple created");
      setOpen(false);
      setForm({
        object_type: "",
        object_id: "",
        relation: "",
        subject_type: "",
        subject_id: "",
        subject_relation: "",
      });
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to create tuple");
    }
  };

  const updateField = (field: string, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const isValid =
    form.object_type &&
    form.object_id &&
    form.relation &&
    form.subject_type &&
    form.subject_id;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Create Tuple
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Relation Tuple</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Object Type *</Label>
              <Input
                placeholder="document"
                value={form.object_type}
                onChange={(e) => updateField("object_type", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Object ID *</Label>
              <Input
                placeholder="doc_1"
                value={form.object_id}
                onChange={(e) => updateField("object_id", e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label>Relation *</Label>
            <Input
              placeholder="viewer"
              value={form.relation}
              onChange={(e) => updateField("relation", e.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Subject Type *</Label>
              <Input
                placeholder="user"
                value={form.subject_type}
                onChange={(e) => updateField("subject_type", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Subject ID *</Label>
              <Input
                placeholder="alice"
                value={form.subject_id}
                onChange={(e) => updateField("subject_id", e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label>Subject Relation (optional)</Label>
            <Input
              placeholder="member"
              value={form.subject_relation}
              onChange={(e) => updateField("subject_relation", e.target.value)}
            />
          </div>
          <Button type="submit" disabled={!isValid || writeTuple.isPending}>
            {writeTuple.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Create
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}
