"use client";

import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Search, X } from "lucide-react";
import type { TupleFilter } from "@/lib/api/types";

interface TupleFiltersProps {
  filter: TupleFilter;
  onChange: (filter: TupleFilter) => void;
  onClear: () => void;
}

export function TupleFilters({ filter, onChange, onClear }: TupleFiltersProps) {
  const updateField = (field: keyof TupleFilter, value: string) => {
    onChange({ ...filter, [field]: value || undefined });
  };

  const hasFilters = Object.values(filter).some(Boolean);

  return (
    <div className="flex flex-wrap items-end gap-2">
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">Object Type</label>
        <Input
          placeholder="e.g. document"
          value={filter.object_type ?? ""}
          onChange={(e) => updateField("object_type", e.target.value)}
          className="h-8 w-36 text-sm"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">Object ID</label>
        <Input
          placeholder="e.g. doc_1"
          value={filter.object_id ?? ""}
          onChange={(e) => updateField("object_id", e.target.value)}
          className="h-8 w-36 text-sm"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">Relation</label>
        <Input
          placeholder="e.g. viewer"
          value={filter.relation ?? ""}
          onChange={(e) => updateField("relation", e.target.value)}
          className="h-8 w-32 text-sm"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">Subject Type</label>
        <Input
          placeholder="e.g. user"
          value={filter.subject_type ?? ""}
          onChange={(e) => updateField("subject_type", e.target.value)}
          className="h-8 w-32 text-sm"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">Subject ID</label>
        <Input
          placeholder="e.g. alice"
          value={filter.subject_id ?? ""}
          onChange={(e) => updateField("subject_id", e.target.value)}
          className="h-8 w-32 text-sm"
        />
      </div>
      <div className="flex items-center gap-1">
        <Button size="sm" variant="outline" className="h-8">
          <Search className="h-3.5 w-3.5" />
        </Button>
        {hasFilters && (
          <Button
            size="sm"
            variant="ghost"
            className="h-8"
            onClick={onClear}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>
    </div>
  );
}
