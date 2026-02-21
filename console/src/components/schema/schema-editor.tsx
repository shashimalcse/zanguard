"use client";

import { useState } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { yaml } from "@codemirror/lang-yaml";
import { Button } from "@/components/ui/button";
import { Save, Upload, Loader2 } from "lucide-react";
import { useSaveSchema } from "@/lib/hooks/use-schema";
import { toast } from "sonner";

interface SchemaEditorProps {
  tenantId: string;
  initialValue: string;
}

export function SchemaEditor({ tenantId, initialValue }: SchemaEditorProps) {
  const [value, setValue] = useState(initialValue);
  const [errors, setErrors] = useState<string[]>([]);
  const saveSchema = useSaveSchema(tenantId);

  const handleSave = async () => {
    setErrors([]);
    try {
      await saveSchema.mutateAsync(value);
      toast.success("Schema saved successfully");
    } catch (err: unknown) {
      const error = err as { details?: string[]; message?: string };
      if (error.details) {
        setErrors(error.details);
      } else {
        toast.error(error.message || "Failed to save schema");
      }
    }
  };

  const handleUpload = () => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".yaml,.yml";
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (file) {
        const text = await file.text();
        setValue(text);
      }
    };
    input.click();
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Button onClick={handleSave} disabled={saveSchema.isPending} size="sm">
          {saveSchema.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          Save Schema
        </Button>
        <Button variant="outline" size="sm" onClick={handleUpload}>
          <Upload className="mr-2 h-4 w-4" />
          Upload File
        </Button>
      </div>

      <div className="rounded-md border overflow-hidden">
        <CodeMirror
          value={value}
          height="500px"
          extensions={[yaml()]}
          onChange={(val) => setValue(val)}
          theme="light"
          basicSetup={{
            lineNumbers: true,
            bracketMatching: true,
            indentOnInput: true,
            foldGutter: true,
          }}
        />
      </div>

      {errors.length > 0 && (
        <div className="rounded-md border border-destructive/50 bg-destructive/5 p-4">
          <p className="text-sm font-medium text-destructive mb-2">
            Schema validation errors:
          </p>
          <ul className="list-disc list-inside space-y-1">
            {errors.map((err, i) => (
              <li key={i} className="text-sm text-destructive">
                {err}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
