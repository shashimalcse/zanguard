"use client";

import { useState, useMemo } from "react";
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
import { Play, Loader2, CheckCircle, XCircle } from "lucide-react";
import { useCheck } from "@/lib/hooks/use-check";
import { cn } from "@/lib/utils";

// ---- Schema parser ----

interface SchemaTypeInfo {
  name: string;
  permissions: string[];
}

function parseSchemaTypeInfos(source: string): SchemaTypeInfo[] {
  const lines = source.split("\n");
  const result: SchemaTypeInfo[] = [];

  let inTypesSection = false;
  let currentType: SchemaTypeInfo | null = null;
  let currentSection: "permissions" | null = null;

  for (const rawLine of lines) {
    if (!rawLine.trim() || rawLine.trimStart().startsWith("#")) continue;
    const indent = rawLine.length - rawLine.trimStart().length;
    const line = rawLine.trim();

    if (indent === 0) {
      inTypesSection = line === "types:";
      currentType = null;
      currentSection = null;
      continue;
    }

    if (!inTypesSection) continue;

    if (indent === 2 && line.endsWith(":")) {
      const typeName = line.slice(0, -1).trim();
      if (!typeName) continue;
      currentType = { name: typeName, permissions: [] };
      result.push(currentType);
      currentSection = null;
      continue;
    }

    if (!currentType) continue;

    if (indent === 4 && line.endsWith(":")) {
      currentSection =
        line.slice(0, -1).trim() === "permissions" ? "permissions" : null;
      continue;
    }

    if (currentSection === "permissions" && indent === 6 && line.endsWith(":")) {
      currentType.permissions.push(line.slice(0, -1).trim());
    }
  }

  return result;
}

// ---- Component ----

interface SchemaEvaluatorProps {
  tenantId: string;
  source: string;
}

type Decision = "allow" | "deny" | null;

export function SchemaEvaluator({ tenantId, source }: SchemaEvaluatorProps) {
  const [form, setForm] = useState({
    resource_type: "",
    resource_id: "",
    action: "",
    subject_type: "",
    subject_id: "",
  });
  const [decision, setDecision] = useState<Decision>(null);
  const check = useCheck(tenantId);

  const schemaTypes = useMemo(
    () => parseSchemaTypeInfos(source),
    [source]
  );

  const typeNames = useMemo(() => schemaTypes.map((t) => t.name), [schemaTypes]);

  const actionsForType = useMemo(() => {
    if (!form.resource_type) return [];
    return (
      schemaTypes.find((t) => t.name === form.resource_type)?.permissions ?? []
    );
  }, [schemaTypes, form.resource_type]);

  const handleCheck = async () => {
    setDecision(null);
    try {
      const result = await check.mutateAsync({
        resource_type: form.resource_type,
        resource_id: form.resource_id,
        action: form.action,
        subject_type: form.subject_type,
        subject_id: form.subject_id,
      });
      setDecision(result.decision ? "allow" : "deny");
    } catch {
      setDecision("deny");
    }
  };

  const update = (field: string, value: string) =>
    setForm((prev) => ({ ...prev, [field]: value }));

  const isValid =
    form.resource_type &&
    form.resource_id &&
    form.action &&
    form.subject_type &&
    form.subject_id;

  const hasSchema = typeNames.length > 0;

  return (
    <div className="space-y-6 max-w-xl">
      <div className="space-y-4">
        {/* Subject */}
        <div>
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
            Subject
          </p>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm">Type *</Label>
              {hasSchema ? (
                <Select
                  value={form.subject_type}
                  onValueChange={(v) => update("subject_type", v)}
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
                  placeholder="user"
                  value={form.subject_type}
                  onChange={(e) => update("subject_type", e.target.value)}
                />
              )}
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm">ID *</Label>
              <Input
                placeholder="alice"
                value={form.subject_id}
                onChange={(e) => update("subject_id", e.target.value)}
              />
            </div>
          </div>
        </div>

        {/* Action */}
        <div>
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
            Action
          </p>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm">Resource Type *</Label>
              {hasSchema ? (
                <Select
                  value={form.resource_type}
                  onValueChange={(v) =>
                    setForm((prev) => ({
                      ...prev,
                      resource_type: v,
                      action: "",
                    }))
                  }
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
                  value={form.resource_type}
                  onChange={(e) => update("resource_type", e.target.value)}
                />
              )}
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm">Permission *</Label>
              {actionsForType.length > 0 ? (
                <Select
                  value={form.action}
                  onValueChange={(v) => update("action", v)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select permission" />
                  </SelectTrigger>
                  <SelectContent>
                    {actionsForType.map((perm) => (
                      <SelectItem key={perm} value={perm}>
                        {perm}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  placeholder="read"
                  value={form.action}
                  onChange={(e) => update("action", e.target.value)}
                  disabled={hasSchema && !form.resource_type}
                />
              )}
            </div>
          </div>
          <div className="mt-3 space-y-1.5">
            <Label className="text-sm">Resource ID *</Label>
            <Input
              placeholder="doc_1"
              value={form.resource_id}
              onChange={(e) => update("resource_id", e.target.value)}
            />
          </div>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <Button
          onClick={handleCheck}
          disabled={!isValid || check.isPending}
        >
          {check.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Play className="mr-2 h-4 w-4" />
          )}
          Check
        </Button>

        {decision && (
          <div
            className={cn(
              "flex items-center gap-2 rounded-md px-4 py-2 text-sm font-semibold",
              decision === "allow"
                ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                : "bg-red-50 text-red-700 border border-red-200"
            )}
          >
            {decision === "allow" ? (
              <CheckCircle className="h-4 w-4" />
            ) : (
              <XCircle className="h-4 w-4" />
            )}
            {decision === "allow" ? "ALLOW" : "DENY"}
          </div>
        )}
      </div>
    </div>
  );
}
