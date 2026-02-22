"use client";

import { useState, useMemo } from "react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Plus, Loader2 } from "lucide-react";
import { useWriteTuple } from "@/lib/hooks/use-tuples";
import { useSchema } from "@/lib/hooks/use-schema";
import { toast } from "sonner";

// ---- Schema parser ----

interface SchemaRelation {
  name: string;
  allowedTypes: string[];
}

interface SchemaTypeInfo {
  name: string;
  relations: SchemaRelation[];
}

function parseSchemaTypes(source: string): SchemaTypeInfo[] {
  const lines = source.split("\n");
  const result: SchemaTypeInfo[] = [];

  let inTypesSection = false;
  let currentType: SchemaTypeInfo | null = null;
  let currentSection: "relations" | null = null;
  let currentRelation: SchemaRelation | null = null;
  let collectingTypesList = false;

  function stripQuotes(v: string) {
    return v.replace(/^['\"]|['\"]$/g, "");
  }

  function parseInlineList(value: string): string[] {
    const normalized = value.trim();
    if (!normalized) return [];
    if (normalized.startsWith("[") && normalized.endsWith("]")) {
      return normalized
        .slice(1, -1)
        .split(",")
        .map((item) => stripQuotes(item.trim()))
        .filter(Boolean);
    }
    return [stripQuotes(normalized)];
  }

  for (const rawLine of lines) {
    if (!rawLine.trim() || rawLine.trimStart().startsWith("#")) continue;
    const indent = rawLine.length - rawLine.trimStart().length;
    const line = rawLine.trim();

    if (indent === 0) {
      inTypesSection = line === "types:";
      currentType = null;
      currentSection = null;
      currentRelation = null;
      collectingTypesList = false;
      continue;
    }

    if (!inTypesSection) continue;

    if (indent === 2 && line.endsWith(":")) {
      const typeName = line.slice(0, -1).trim();
      if (!typeName) continue;
      currentType = { name: typeName, relations: [] };
      result.push(currentType);
      currentSection = null;
      currentRelation = null;
      collectingTypesList = false;
      continue;
    }

    if (!currentType) continue;

    if (indent === 4 && line.endsWith(":")) {
      currentSection =
        line.slice(0, -1).trim() === "relations" ? "relations" : null;
      currentRelation = null;
      collectingTypesList = false;
      continue;
    }

    if (currentSection !== "relations") continue;

    if (indent === 6 && line.endsWith(":")) {
      currentRelation = { name: line.slice(0, -1).trim(), allowedTypes: [] };
      currentType.relations.push(currentRelation);
      collectingTypesList = false;
      continue;
    }

    if (!currentRelation) continue;

    if (indent === 8 && line.startsWith("types:")) {
      const rawTypes = line.slice("types:".length).trim();
      if (rawTypes) {
        currentRelation.allowedTypes.push(...parseInlineList(rawTypes));
        collectingTypesList = false;
      } else {
        collectingTypesList = true;
      }
      continue;
    }

    if (collectingTypesList && indent >= 10 && line.startsWith("-")) {
      const item = line.slice(1).trim();
      if (item) currentRelation.allowedTypes.push(stripQuotes(item));
      continue;
    }

    if (indent <= 8) collectingTypesList = false;
  }

  return result;
}

// ---- Component ----

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
  const { data: schema } = useSchema(tenantId);

  const schemaTypes = useMemo(() => {
    if (!schema?.source) return [];
    return parseSchemaTypes(schema.source);
  }, [schema?.source]);

  const typeNames = useMemo(() => schemaTypes.map((t) => t.name), [schemaTypes]);

  const relationsForType = useMemo(() => {
    if (!form.object_type) return [];
    return (
      schemaTypes.find((t) => t.name === form.object_type)?.relations ?? []
    );
  }, [schemaTypes, form.object_type]);

  const subjectTypesForRelation = useMemo(() => {
    const rel = relationsForType.find((r) => r.name === form.relation);
    if (rel && rel.allowedTypes.length > 0) {
      return [
        ...new Set(
          rel.allowedTypes.map((t) => t.split("#")[0]).filter(Boolean)
        ),
      ];
    }
    return typeNames;
  }, [relationsForType, form.relation, typeNames]);

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

  const hasSchema = typeNames.length > 0;

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
              {hasSchema ? (
                <Select
                  value={form.object_type}
                  onValueChange={(v) => {
                    const rels =
                      schemaTypes.find((t) => t.name === v)?.relations ?? [];
                    const autoRel = rels.length === 1 ? rels[0] : null;
                    const allowed = autoRel
                      ? autoRel.allowedTypes.length > 0
                        ? [...new Set(autoRel.allowedTypes.map((t) => t.split("#")[0]).filter(Boolean))]
                        : typeNames
                      : [];
                    setForm((prev) => ({
                      ...prev,
                      object_type: v,
                      relation: autoRel ? autoRel.name : "",
                      subject_type: allowed.length === 1 ? allowed[0] : "",
                    }));
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select type" />
                  </SelectTrigger>
                  <SelectContent>
                    {typeNames.map((name) => (
                      <SelectItem key={name} value={name}>
                        {name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  placeholder="document"
                  value={form.object_type}
                  onChange={(e) => updateField("object_type", e.target.value)}
                />
              )}
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
            {relationsForType.length > 0 ? (
              <Select
                value={form.relation}
                onValueChange={(v) => {
                  const rel = relationsForType.find((r) => r.name === v);
                  const allowed =
                    rel && rel.allowedTypes.length > 0
                      ? [...new Set(rel.allowedTypes.map((t) => t.split("#")[0]).filter(Boolean))]
                      : typeNames;
                  setForm((prev) => ({
                    ...prev,
                    relation: v,
                    subject_type: allowed.length === 1 ? allowed[0] : "",
                  }));
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select relation" />
                </SelectTrigger>
                <SelectContent>
                  {relationsForType.map((rel) => (
                    <SelectItem key={rel.name} value={rel.name}>
                      {rel.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <Input
                placeholder="viewer"
                value={form.relation}
                onChange={(e) => updateField("relation", e.target.value)}
                disabled={hasSchema && !form.object_type}
              />
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Subject Type *</Label>
              {subjectTypesForRelation.length > 0 ? (
                <Select
                  value={form.subject_type}
                  onValueChange={(v) => updateField("subject_type", v)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select type" />
                  </SelectTrigger>
                  <SelectContent>
                    {subjectTypesForRelation.map((name) => (
                      <SelectItem key={name} value={name}>
                        {name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  placeholder="user"
                  value={form.subject_type}
                  onChange={(e) => updateField("subject_type", e.target.value)}
                />
              )}
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
            <Label>
              Subject Relation{" "}
              <span className="text-muted-foreground text-xs">(optional)</span>
            </Label>
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
