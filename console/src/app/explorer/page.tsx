"use client";

import { useState } from "react";
import { Network, Building2, Loader2, Play } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { useTenantStore } from "@/lib/stores/tenant-store";
import { useExpand } from "@/lib/hooks/use-expand";
import { SubjectTreeView } from "@/components/explorer/subject-tree";
import { EmptyState } from "@/components/shared/empty-state";
import { toast } from "sonner";
import type { SubjectTree } from "@/lib/api/types";

export default function ExplorerPage() {
  const { selectedTenantId } = useTenantStore();
  const [form, setForm] = useState({
    object_type: "",
    object_id: "",
    relation: "",
  });
  const [tree, setTree] = useState<SubjectTree | null>(null);
  const expand = useExpand(selectedTenantId ?? "");

  if (!selectedTenantId) {
    return (
      <EmptyState
        icon={Building2}
        title="No tenant selected"
        description="Select a tenant from the dropdown above to explore permission trees."
      />
    );
  }

  const handleExpand = async () => {
    try {
      const result = await expand.mutateAsync({
        object_type: form.object_type,
        object_id: form.object_id,
        relation: form.relation,
      });
      setTree(result);
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to expand relation");
    }
  };

  const isValid = form.object_type && form.object_id && form.relation;

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Explorer</h1>

      <div className="flex items-end gap-2">
        <div className="space-y-1">
          <Label className="text-xs">Object Type</Label>
          <Input
            placeholder="document"
            value={form.object_type}
            onChange={(e) =>
              setForm((f) => ({ ...f, object_type: e.target.value }))
            }
            className="h-8 w-40 text-sm"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Object ID</Label>
          <Input
            placeholder="doc_1"
            value={form.object_id}
            onChange={(e) =>
              setForm((f) => ({ ...f, object_id: e.target.value }))
            }
            className="h-8 w-40 text-sm"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Relation</Label>
          <Input
            placeholder="viewer"
            value={form.relation}
            onChange={(e) =>
              setForm((f) => ({ ...f, relation: e.target.value }))
            }
            className="h-8 w-40 text-sm"
          />
        </div>
        <Button
          size="sm"
          className="h-8"
          onClick={handleExpand}
          disabled={!isValid || expand.isPending}
        >
          {expand.isPending ? (
            <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
          ) : (
            <Play className="mr-2 h-3.5 w-3.5" />
          )}
          Expand
        </Button>
      </div>

      {tree ? (
        <SubjectTreeView tree={tree} />
      ) : (
        <EmptyState
          icon={Network}
          title="No tree to display"
          description="Enter an object type, ID, and relation, then click Expand to visualize the subject tree."
        />
      )}
    </div>
  );
}
