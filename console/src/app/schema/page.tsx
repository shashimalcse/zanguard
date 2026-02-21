"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FileCode, Building2 } from "lucide-react";
import { useSchema } from "@/lib/hooks/use-schema";
import { useTenantStore } from "@/lib/stores/tenant-store";
import { SchemaEditor } from "@/components/schema/schema-editor";
import { SchemaViewer } from "@/components/schema/schema-viewer";
import { SchemaMeta } from "@/components/schema/schema-meta";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";

export default function SchemaPage() {
  const { selectedTenantId } = useTenantStore();
  const { data: schema, isLoading, error } = useSchema(selectedTenantId);

  if (!selectedTenantId) {
    return (
      <EmptyState
        icon={Building2}
        title="No tenant selected"
        description="Select a tenant from the dropdown above to manage its authorization schema."
      />
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-[500px] w-full" />
      </div>
    );
  }

  const hasSchema = schema && !error;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Schema</h1>
      </div>

      {hasSchema && (
        <SchemaMeta
          hash={schema.hash}
          version={schema.version}
          compiledAt={schema.compiled_at}
        />
      )}

      <Tabs defaultValue="editor">
        <TabsList>
          <TabsTrigger value="editor">
            <FileCode className="mr-2 h-4 w-4" />
            Editor
          </TabsTrigger>
          <TabsTrigger value="visual" disabled={!hasSchema}>
            Visual
          </TabsTrigger>
        </TabsList>

        <TabsContent value="editor" className="mt-4">
          <SchemaEditor
            tenantId={selectedTenantId}
            initialValue={schema?.source ?? ""}
          />
        </TabsContent>

        <TabsContent value="visual" className="mt-4">
          {hasSchema ? (
            <SchemaViewer source={schema.source} />
          ) : (
            <EmptyState
              icon={FileCode}
              title="No schema loaded"
              description="Load a schema using the Editor tab to see the visual representation."
            />
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
