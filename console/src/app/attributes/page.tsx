"use client";

import { useState } from "react";
import { Tags, Building2, Search, Loader2 } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useTenantStore } from "@/lib/stores/tenant-store";
import {
  useObjectAttributes,
  useSubjectAttributes,
  useSaveObjectAttributes,
  useSaveSubjectAttributes,
} from "@/lib/hooks/use-attributes";
import { AttributeEditor } from "@/components/attributes/attribute-editor";
import { EmptyState } from "@/components/shared/empty-state";
import { toast } from "sonner";

function ObjectAttributesTab({ tenantId }: { tenantId: string }) {
  const [type, setType] = useState("");
  const [id, setId] = useState("");
  const [query, setQuery] = useState({ type: "", id: "" });

  const { data, isLoading, error } = useObjectAttributes(
    tenantId,
    query.type,
    query.id
  );
  const saveAttrs = useSaveObjectAttributes(tenantId, query.type, query.id);

  const handleLookup = () => {
    setQuery({ type, id });
  };

  const handleSave = async (attributes: Record<string, unknown>) => {
    try {
      await saveAttrs.mutateAsync(attributes);
      toast.success("Object attributes saved");
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to save attributes");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-end gap-2">
        <div className="space-y-1">
          <Label className="text-xs">Object Type</Label>
          <Input
            placeholder="document"
            value={type}
            onChange={(e) => setType(e.target.value)}
            className="h-8 w-40 text-sm"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Object ID</Label>
          <Input
            placeholder="doc_1"
            value={id}
            onChange={(e) => setId(e.target.value)}
            className="h-8 w-40 text-sm"
          />
        </div>
        <Button
          size="sm"
          className="h-8"
          onClick={handleLookup}
          disabled={!type || !id}
        >
          <Search className="mr-2 h-3.5 w-3.5" />
          Lookup
        </Button>
      </div>

      {isLoading && (
        <div className="flex items-center gap-2 py-8 justify-center text-muted-foreground text-sm">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading...
        </div>
      )}

      {query.type && query.id && !isLoading && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-mono">
              {query.type}:{query.id}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <AttributeEditor
              attributes={data?.attributes ?? {}}
              onSave={handleSave}
              isSaving={saveAttrs.isPending}
            />
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function SubjectAttributesTab({ tenantId }: { tenantId: string }) {
  const [type, setType] = useState("");
  const [id, setId] = useState("");
  const [query, setQuery] = useState({ type: "", id: "" });

  const { data, isLoading } = useSubjectAttributes(
    tenantId,
    query.type,
    query.id
  );
  const saveAttrs = useSaveSubjectAttributes(tenantId, query.type, query.id);

  const handleLookup = () => {
    setQuery({ type, id });
  };

  const handleSave = async (attributes: Record<string, unknown>) => {
    try {
      await saveAttrs.mutateAsync(attributes);
      toast.success("Subject attributes saved");
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to save attributes");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-end gap-2">
        <div className="space-y-1">
          <Label className="text-xs">Subject Type</Label>
          <Input
            placeholder="user"
            value={type}
            onChange={(e) => setType(e.target.value)}
            className="h-8 w-40 text-sm"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Subject ID</Label>
          <Input
            placeholder="alice"
            value={id}
            onChange={(e) => setId(e.target.value)}
            className="h-8 w-40 text-sm"
          />
        </div>
        <Button
          size="sm"
          className="h-8"
          onClick={handleLookup}
          disabled={!type || !id}
        >
          <Search className="mr-2 h-3.5 w-3.5" />
          Lookup
        </Button>
      </div>

      {isLoading && (
        <div className="flex items-center gap-2 py-8 justify-center text-muted-foreground text-sm">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading...
        </div>
      )}

      {query.type && query.id && !isLoading && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-mono">
              {query.type}:{query.id}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <AttributeEditor
              attributes={data?.attributes ?? {}}
              onSave={handleSave}
              isSaving={saveAttrs.isPending}
            />
          </CardContent>
        </Card>
      )}
    </div>
  );
}

export default function AttributesPage() {
  const { selectedTenantId } = useTenantStore();

  if (!selectedTenantId) {
    return (
      <EmptyState
        icon={Building2}
        title="No tenant selected"
        description="Select a tenant from the dropdown above to manage attributes."
      />
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Attributes</h1>

      <Tabs defaultValue="objects">
        <TabsList>
          <TabsTrigger value="objects">Object Attributes</TabsTrigger>
          <TabsTrigger value="subjects">Subject Attributes</TabsTrigger>
        </TabsList>

        <TabsContent value="objects" className="mt-4">
          <ObjectAttributesTab tenantId={selectedTenantId} />
        </TabsContent>

        <TabsContent value="subjects" className="mt-4">
          <SubjectAttributesTab tenantId={selectedTenantId} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
