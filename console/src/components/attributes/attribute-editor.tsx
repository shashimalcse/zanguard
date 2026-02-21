"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Plus, Trash2, Save, Loader2 } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface AttributeEditorProps {
  attributes: Record<string, unknown>;
  onSave: (attributes: Record<string, unknown>) => Promise<void>;
  isSaving: boolean;
}

export function AttributeEditor({
  attributes,
  onSave,
  isSaving,
}: AttributeEditorProps) {
  const [entries, setEntries] = useState<{ key: string; value: string }[]>(
    Object.entries(attributes).map(([key, value]) => ({
      key,
      value: typeof value === "string" ? value : JSON.stringify(value),
    }))
  );

  const addEntry = () => {
    setEntries([...entries, { key: "", value: "" }]);
  };

  const removeEntry = (index: number) => {
    setEntries(entries.filter((_, i) => i !== index));
  };

  const updateEntry = (
    index: number,
    field: "key" | "value",
    val: string
  ) => {
    const updated = [...entries];
    updated[index] = { ...updated[index], [field]: val };
    setEntries(updated);
  };

  const handleSave = () => {
    const attrs: Record<string, unknown> = {};
    for (const entry of entries) {
      if (entry.key.trim()) {
        try {
          attrs[entry.key.trim()] = JSON.parse(entry.value);
        } catch {
          attrs[entry.key.trim()] = entry.value;
        }
      }
    }
    onSave(attrs);
  };

  return (
    <div className="space-y-3">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[200px]">Key</TableHead>
            <TableHead>Value</TableHead>
            <TableHead className="w-[40px]" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.map((entry, i) => (
            <TableRow key={i}>
              <TableCell>
                <Input
                  value={entry.key}
                  onChange={(e) => updateEntry(i, "key", e.target.value)}
                  placeholder="key"
                  className="h-8 text-sm"
                />
              </TableCell>
              <TableCell>
                <Input
                  value={entry.value}
                  onChange={(e) => updateEntry(i, "value", e.target.value)}
                  placeholder="value"
                  className="h-8 text-sm font-mono"
                />
              </TableCell>
              <TableCell>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0"
                  onClick={() => removeEntry(i)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={addEntry}>
          <Plus className="mr-2 h-4 w-4" />
          Add Attribute
        </Button>
        <Button size="sm" onClick={handleSave} disabled={isSaving}>
          {isSaving ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          Save
        </Button>
      </div>
    </div>
  );
}
